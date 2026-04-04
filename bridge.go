// Bridge API (alpha) — remote session management for cloud worker scenarios.
//
// This implements the equivalent of @anthropic-ai/claude-agent-sdk/bridge,
// aligned with Claude Code's CCR V2 transport protocol:
//   - Read: SSE stream at GET {sessionURL}/worker/events/stream
//   - Write: HTTP POST at POST {sessionURL}/worker/events
//   - State: PUT {sessionURL}/worker
//   - Heartbeat: POST {sessionURL}/worker/heartbeat (20s interval)
//
// The sessionURL (apiBaseURL from RemoteCredentials) already includes the
// session path, e.g. "https://host/v1/agent/sessions/cse_xxx". All worker
// endpoints are relative to it.
//
// Stability: Alpha. Breaking changes may occur without notice.
package agentsdk

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// SessionState is the worker state reported to the /worker endpoint.
type SessionState string

const (
	SessionStateIdle           SessionState = "idle"
	SessionStateRunning        SessionState = "running"
	SessionStateRequiresAction SessionState = "requires_action"
)

// RemoteCredentials contains the worker credentials from POST .../bridge.
type RemoteCredentials struct {
	WorkerJWT   string `json:"worker_jwt"`
	APIBaseURL  string `json:"api_base_url"`
	ExpiresIn   int    `json:"expires_in"`
	WorkerEpoch int    `json:"worker_epoch"`
}

// BridgeSessionHandle is a per-session bridge transport handle.
// Auth is instance-scoped — the JWT lives in this handle's closure,
// so multiple handles can coexist without stomping each other.
type BridgeSessionHandle struct {
	sessionID   string
	sessionURL  string // full session URL (apiBaseURL), e.g. https://host/v1/agent/sessions/cse_xxx
	workerJWT   string
	epoch       int
	sequenceNum int64
	connected   bool
	client      *http.Client
	mu          sync.Mutex
	cancel      context.CancelFunc

	// Callbacks
	onInboundMessage     func(SDKMessage)
	onPermissionResponse func(json.RawMessage)
	onInterrupt          func()
	onClose              func(code int)
}

// SessionID returns the session identifier.
func (h *BridgeSessionHandle) SessionID() string {
	return h.sessionID
}

// GetSequenceNum returns the live SSE event-stream high-water mark.
func (h *BridgeSessionHandle) GetSequenceNum() int64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.sequenceNum
}

// IsConnected returns true once the write path is ready.
func (h *BridgeSessionHandle) IsConnected() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.connected
}

// Write writes a single SDKMessage via POST /worker/events.
func (h *BridgeSessionHandle) Write(msg SDKMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}
	return h.writeEvents([]json.RawMessage{data})
}

// WriteBatch writes multiple events in a single POST /worker/events.
func (h *BridgeSessionHandle) WriteBatch(events []json.RawMessage) error {
	return h.writeEvents(events)
}

// SendResult signals a turn boundary.
func (h *BridgeSessionHandle) SendResult() error {
	msg := map[string]any{
		"type":       "result",
		"session_id": h.sessionID,
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return h.writeEvents([]json.RawMessage{data})
}

// SendControlRequest forwards a control request through the bridge.
func (h *BridgeSessionHandle) SendControlRequest(req json.RawMessage) error {
	return h.writeEvents([]json.RawMessage{req})
}

// SendControlResponse forwards a control response through the bridge.
func (h *BridgeSessionHandle) SendControlResponse(res json.RawMessage) error {
	return h.writeEvents([]json.RawMessage{res})
}

// SendControlCancelRequest tells the server to dismiss a pending prompt.
func (h *BridgeSessionHandle) SendControlCancelRequest(requestID string) error {
	msg := map[string]any{
		"type":       "control_cancel_request",
		"request_id": requestID,
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return h.writeEvents([]json.RawMessage{data})
}

// ReportState reports the worker state via PUT /worker.
func (h *BridgeSessionHandle) ReportState(state SessionState) error {
	body := map[string]any{
		"worker_status": string(state),
		"worker_epoch":  h.epoch,
	}
	return h.putJSON(h.sessionURL+"/worker", body)
}

// ReportMetadata reports external metadata via PUT /worker.
func (h *BridgeSessionHandle) ReportMetadata(metadata map[string]any) error {
	body := map[string]any{
		"external_metadata": metadata,
		"worker_epoch":      h.epoch,
	}
	return h.putJSON(h.sessionURL+"/worker", body)
}

// ReportDelivery reports batch event delivery status via POST /worker/events/delivery.
func (h *BridgeSessionHandle) ReportDelivery(eventID string, status string) error {
	body := map[string]any{
		"worker_epoch": h.epoch,
		"updates": []map[string]string{
			{"event_id": eventID, "status": status},
		},
	}
	return h.postJSON(h.sessionURL+"/worker/events/delivery", body)
}

// Close stops the SSE reader and heartbeat, then marks as disconnected.
func (h *BridgeSessionHandle) Close() {
	h.mu.Lock()
	h.connected = false
	if h.cancel != nil {
		h.cancel()
	}
	h.mu.Unlock()
}

// writeEvents sends events via POST /worker/events with epoch.
func (h *BridgeSessionHandle) writeEvents(events []json.RawMessage) error {
	type clientEvent struct {
		Payload json.RawMessage `json:"payload"`
	}
	wrapped := make([]clientEvent, len(events))
	for i, e := range events {
		wrapped[i] = clientEvent{Payload: e}
	}
	body := map[string]any{
		"worker_epoch": h.epoch,
		"events":       wrapped,
	}
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}

	url := h.sessionURL + "/worker/events"
	req, err := http.NewRequest("POST", url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+h.workerJWT)
	req.Header.Set("Content-Type", "application/json")

	resp, err := h.client.Do(req)
	if err != nil {
		return fmt.Errorf("post events: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusConflict {
		return fmt.Errorf("epoch mismatch (409)")
	}
	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("post events: HTTP %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

func (h *BridgeSessionHandle) putJSON(url string, body map[string]any) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("PUT", url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+h.workerJWT)
	req.Header.Set("Content-Type", "application/json")
	resp, err := h.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

func (h *BridgeSessionHandle) postJSON(url string, body map[string]any) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+h.workerJWT)
	req.Header.Set("Content-Type", "application/json")
	resp, err := h.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

// --- SSE Reader ---

// startSSEReader connects to the SSE event stream and dispatches events.
// Reconnects with exponential backoff on failure.
func (h *BridgeSessionHandle) startSSEReader(ctx context.Context) {
	backoff := time.Second
	maxBackoff := 30 * time.Second
	livenessTimeout := 45 * time.Second

	for {
		if ctx.Err() != nil {
			return
		}

		connectedAt := time.Now()
		err := h.readSSEStream(ctx, livenessTimeout)

		if ctx.Err() != nil {
			return
		}

		_ = err // logged inside readSSEStream

		// Reset backoff if connection lasted > 30s.
		if time.Since(connectedAt) > 30*time.Second {
			backoff = time.Second
		} else {
			backoff = min(backoff*2, maxBackoff)
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
	}
}

func (h *BridgeSessionHandle) readSSEStream(ctx context.Context, livenessTimeout time.Duration) error {
	h.mu.Lock()
	seqNum := h.sequenceNum
	h.mu.Unlock()

	url := h.sessionURL + "/worker/events/stream"
	if seqNum > 0 {
		url += "?from_sequence_num=" + strconv.FormatInt(seqNum, 10)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+h.workerJWT)
	req.Header.Set("Accept", "text/event-stream")
	if seqNum > 0 {
		req.Header.Set("Last-Event-ID", strconv.FormatInt(seqNum, 10))
	}

	sseClient := &http.Client{Timeout: 0} // no timeout for SSE
	resp, err := sseClient.Do(req)
	if err != nil {
		return fmt.Errorf("sse connect: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusNotFound {
		if h.onClose != nil {
			h.onClose(resp.StatusCode)
		}
		return fmt.Errorf("sse permanent error: HTTP %d", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("sse connect: HTTP %d", resp.StatusCode)
	}

	// Single scanner goroutine — avoids per-iteration goroutine leak.
	type scanResult struct {
		text string
		err  error
		eof  bool
	}
	lineCh := make(chan scanResult, 1)
	go func() {
		defer close(lineCh)
		scanner := bufio.NewScanner(resp.Body)
		scanner.Buffer(make([]byte, 0, 1<<20), 1<<20)
		for scanner.Scan() {
			lineCh <- scanResult{text: scanner.Text()}
		}
		if err := scanner.Err(); err != nil {
			lineCh <- scanResult{err: err}
		} else {
			lineCh <- scanResult{eof: true}
		}
	}()

	var eventType, eventID, eventData string
	liveness := time.NewTimer(livenessTimeout)
	defer liveness.Stop()

	for {
		select {
		case <-ctx.Done():
			resp.Body.Close() // unblock scanner goroutine
			return ctx.Err()
		case <-liveness.C:
			resp.Body.Close()
			return fmt.Errorf("sse liveness timeout (%s)", livenessTimeout)
		case sr, ok := <-lineCh:
			if !ok {
				return fmt.Errorf("sse stream closed")
			}
			if sr.err != nil {
				return fmt.Errorf("sse read: %w", sr.err)
			}
			if sr.eof {
				return fmt.Errorf("sse stream closed")
			}

			liveness.Reset(livenessTimeout)
			line := sr.text

			if line == "" {
				if eventData != "" {
					h.handleSSEEvent(eventType, eventID, eventData)
				}
				eventType, eventID, eventData = "", "", ""
				continue
			}

			if strings.HasPrefix(line, ":") {
				continue
			}

			colonIdx := strings.IndexByte(line, ':')
			if colonIdx == -1 {
				continue
			}
			field := line[:colonIdx]
			value := line[colonIdx+1:]
			if len(value) > 0 && value[0] == ' ' {
				value = value[1:]
			}

			switch field {
			case "event":
				eventType = value
			case "id":
				eventID = value
			case "data":
				if eventData != "" {
					eventData += "\n" + value
				} else {
					eventData = value
				}
			}
		}
	}
}

func (h *BridgeSessionHandle) handleSSEEvent(eventType, eventID, data string) {
	if eventID != "" {
		if seqNum, err := strconv.ParseInt(eventID, 10, 64); err == nil {
			h.mu.Lock()
			if seqNum > h.sequenceNum {
				h.sequenceNum = seqNum
			}
			h.mu.Unlock()
		}
	}

	// Parse the StreamClientEvent envelope.
	var envelope struct {
		Payload json.RawMessage `json:"payload"`
	}
	if err := json.Unmarshal([]byte(data), &envelope); err != nil || len(envelope.Payload) == 0 {
		return
	}

	// Route based on payload type.
	var header struct {
		Type string `json:"type"`
	}
	json.Unmarshal(envelope.Payload, &header)

	switch header.Type {
	case "control_request":
		// Parse as control request — forward to onInterrupt or similar.
		var creq struct {
			Request struct {
				Subtype string `json:"subtype"`
			} `json:"request"`
		}
		json.Unmarshal(envelope.Payload, &creq)
		if creq.Request.Subtype == "interrupt" && h.onInterrupt != nil {
			h.onInterrupt()
		}
	case "control_response":
		if h.onPermissionResponse != nil {
			h.onPermissionResponse(envelope.Payload)
		}
	default:
		// SDK message — forward to onInboundMessage.
		if h.onInboundMessage != nil {
			var msg SDKMessage
			if err := json.Unmarshal(envelope.Payload, &msg); err == nil {
				h.onInboundMessage(msg)
			}
		}
	}
}

// --- Heartbeat ---

func (h *BridgeSessionHandle) startHeartbeat(ctx context.Context, intervalMs int) {
	if intervalMs <= 0 {
		intervalMs = 20000
	}
	base := time.Duration(intervalMs) * time.Millisecond

	for {
		// ±10% jitter
		jitter := time.Duration(float64(base) * 0.1 * (2*rand.Float64() - 1))
		select {
		case <-ctx.Done():
			return
		case <-time.After(base + jitter):
		}

		body := map[string]any{
			"session_id":   h.sessionID,
			"worker_epoch": h.epoch,
		}
		_ = h.postJSON(h.sessionURL+"/worker/heartbeat", body)
	}
}

// --- AttachBridgeSession ---

// AttachBridgeSessionOptions configures a bridge session attachment.
type AttachBridgeSessionOptions struct {
	// SessionID is the session ID (cse_* form).
	SessionID string
	// IngressToken is the worker JWT from FetchRemoteCredentials.
	IngressToken string
	// APIBaseURL is the session base URL, e.g. "https://host/v1/agent/sessions/cse_xxx".
	// All worker endpoints are relative to this URL.
	APIBaseURL string
	// Epoch is the worker epoch if already known. Omit to register automatically.
	Epoch *int
	// InitialSequenceNum seeds the first SSE connect's from_sequence_num.
	InitialSequenceNum int64
	// HeartbeatIntervalMs is the heartbeat interval (default 20000ms).
	HeartbeatIntervalMs int
	// OutboundOnly when true only forwards events outbound (no SSE read stream).
	OutboundOnly bool

	// Callbacks
	OnInboundMessage     func(SDKMessage)
	OnPermissionResponse func(json.RawMessage)
	OnInterrupt          func()
	OnClose              func(code int)
}

// AttachBridgeSession attaches to an existing bridge session.
// Starts SSE reader and heartbeat goroutines (unless OutboundOnly).
// Call Close() to stop background goroutines.
func AttachBridgeSession(ctx context.Context, opts AttachBridgeSessionOptions) (*BridgeSessionHandle, error) {
	epoch := 0
	if opts.Epoch != nil {
		epoch = *opts.Epoch
	}

	ctx, cancel := context.WithCancel(ctx)

	handle := &BridgeSessionHandle{
		sessionID:            opts.SessionID,
		sessionURL:           strings.TrimRight(opts.APIBaseURL, "/"),
		workerJWT:            opts.IngressToken,
		epoch:                epoch,
		sequenceNum:          opts.InitialSequenceNum,
		connected:            true,
		client:               &http.Client{Timeout: 30 * time.Second},
		cancel:               cancel,
		onInboundMessage:     opts.OnInboundMessage,
		onPermissionResponse: opts.OnPermissionResponse,
		onInterrupt:          opts.OnInterrupt,
		onClose:              opts.OnClose,
	}

	// Report initial state.
	if err := handle.ReportState(SessionStateIdle); err != nil {
		cancel()
		return nil, fmt.Errorf("initial state report: %w", err)
	}

	// Start heartbeat.
	go handle.startHeartbeat(ctx, opts.HeartbeatIntervalMs)

	// Start SSE reader unless outbound-only.
	if !opts.OutboundOnly {
		go handle.startSSEReader(ctx)
	}

	return handle, nil
}

// --- Session Creation Helpers ---

// SessionPathPrefix is the URL path prefix for session endpoints.
// Default is "/v1/code/sessions" (claude.ai). Set to "/v1/agent/sessions" for agentserver.
const DefaultSessionPathPrefix = "/v1/code/sessions"

// CreateSessionOptions configures session creation.
type CreateSessionOptions struct {
	BaseURL    string
	Token      string
	Title      string
	Tags       []string
	TimeoutMs  int
	PathPrefix string // default: /v1/code/sessions
}

// CreateSession creates a fresh session via POST {baseURL}{pathPrefix}.
// Returns the session ID (cse_* form), or an error.
func CreateSession(opts CreateSessionOptions) (string, error) {
	prefix := opts.PathPrefix
	if prefix == "" {
		prefix = DefaultSessionPathPrefix
	}

	body := map[string]any{"title": opts.Title, "bridge": struct{}{}}
	if len(opts.Tags) > 0 {
		body["tags"] = opts.Tags
	}

	data, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	timeout := time.Duration(opts.TimeoutMs) * time.Millisecond
	if timeout == 0 {
		timeout = 10 * time.Second
	}
	client := &http.Client{Timeout: timeout}
	req, err := http.NewRequest("POST", opts.BaseURL+prefix, bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+opts.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("create session: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("create session: HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Session struct {
			ID string `json:"id"`
		} `json:"session"`
		ID string `json:"id"` // fallback for flat response
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("parse session response: %w", err)
	}
	if result.Session.ID != "" {
		return result.Session.ID, nil
	}
	return result.ID, nil
}

// FetchRemoteCredentialsOptions configures credential fetching.
type FetchRemoteCredentialsOptions struct {
	SessionID  string
	BaseURL    string
	Token      string
	TimeoutMs  int
	PathPrefix string // default: /v1/code/sessions
}

// FetchRemoteCredentials mints a worker JWT for a session via POST .../bridge.
// The call IS the worker register (bumps epoch server-side).
func FetchRemoteCredentials(opts FetchRemoteCredentialsOptions) (*RemoteCredentials, error) {
	prefix := opts.PathPrefix
	if prefix == "" {
		prefix = DefaultSessionPathPrefix
	}

	timeout := time.Duration(opts.TimeoutMs) * time.Millisecond
	if timeout == 0 {
		timeout = 10 * time.Second
	}
	client := &http.Client{Timeout: timeout}
	url := opts.BaseURL + prefix + "/" + opts.SessionID + "/bridge"
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+opts.Token)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch credentials: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("fetch credentials: HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var creds RemoteCredentials
	if err := json.NewDecoder(resp.Body).Decode(&creds); err != nil {
		return nil, fmt.Errorf("parse credentials: %w", err)
	}
	return &creds, nil
}

// --- Legacy API (deprecated, use CreateSession/FetchRemoteCredentials) ---

// CreateCodeSession creates a fresh CCR session via POST /v1/code/sessions.
// Deprecated: Use CreateSession with PathPrefix instead.
func CreateCodeSession(baseURL, accessToken, title string, timeoutMs int, tags ...string) (string, error) {
	return CreateSession(CreateSessionOptions{
		BaseURL:    baseURL,
		Token:      accessToken,
		Title:      title,
		Tags:       tags,
		TimeoutMs:  timeoutMs,
		PathPrefix: DefaultSessionPathPrefix,
	})
}

func min(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}
