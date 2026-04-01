// Bridge API (alpha) — remote session management for cloud worker scenarios.
//
// This implements the equivalent of @anthropic-ai/claude-agent-sdk/bridge.
// Use these functions to build cloud-hosted Claude Code workers that connect
// to sessions created via the claude.ai API.
//
// Stability: Alpha. Breaking changes may occur without notice.
package agentsdk

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// SessionState is the worker state reported to the CCR /worker endpoint.
type SessionState string

const (
	SessionStateIdle           SessionState = "idle"
	SessionStateRunning        SessionState = "running"
	SessionStateRequiresAction SessionState = "requires_action"
)

// RemoteCredentials contains the worker credentials from POST /v1/code/sessions/{id}/bridge.
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
	sessionID    string
	apiBaseURL   string
	workerJWT    string
	epoch        int
	sequenceNum  int
	connected    bool
	client       *http.Client
	mu           sync.Mutex

	// Callbacks
	onInboundMessage   func(SDKMessage)
	onPermissionResponse func(json.RawMessage)
	onInterrupt        func()
	onClose            func(code int)
}

// SessionID returns the session identifier.
func (h *BridgeSessionHandle) SessionID() string {
	return h.sessionID
}

// GetSequenceNum returns the live SSE event-stream high-water mark.
// Persist this and pass back as InitialSequenceNum on re-attach so the
// server resumes instead of replaying full history.
func (h *BridgeSessionHandle) GetSequenceNum() int {
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

// Write writes a single SDKMessage. session_id is injected automatically.
func (h *BridgeSessionHandle) Write(msg SDKMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}
	return h.postEvent(data)
}

// SendResult signals a turn boundary — claude.ai stops the "working" spinner.
func (h *BridgeSessionHandle) SendResult() error {
	msg := map[string]any{
		"type":       "result",
		"session_id": h.sessionID,
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return h.postEvent(data)
}

// SendControlRequest forwards a permission request (can_use_tool) to claude.ai.
func (h *BridgeSessionHandle) SendControlRequest(req json.RawMessage) error {
	return h.postEvent(req)
}

// SendControlResponse forwards a permission response back through the bridge.
func (h *BridgeSessionHandle) SendControlResponse(res json.RawMessage) error {
	return h.postEvent(res)
}

// SendControlCancelRequest tells claude.ai to dismiss a pending permission prompt.
func (h *BridgeSessionHandle) SendControlCancelRequest(requestID string) error {
	msg := map[string]any{
		"type":       "control_cancel_request",
		"request_id": requestID,
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return h.postEvent(data)
}

// ReportState reports the worker state to CCR.
// Use "running" on turn start, "requires_action" on permission prompt, "idle" on turn end.
func (h *BridgeSessionHandle) ReportState(state SessionState) error {
	url := fmt.Sprintf("%s/v1/code/worker", h.apiBaseURL)
	body := map[string]any{"state": string(state)}
	return h.putJSON(url, body)
}

// ReportMetadata reports external metadata (branch, dir) shown on claude.ai.
func (h *BridgeSessionHandle) ReportMetadata(metadata map[string]any) error {
	url := fmt.Sprintf("%s/v1/code/worker", h.apiBaseURL)
	body := map[string]any{"external_metadata": metadata}
	return h.putJSON(url, body)
}

// ReportDelivery reports event delivery status.
// Status should be "processing" (turn start) or "processed" (turn end).
func (h *BridgeSessionHandle) ReportDelivery(eventID string, status string) error {
	url := fmt.Sprintf("%s/v1/code/worker/events/%s/delivery", h.apiBaseURL, eventID)
	body := map[string]any{"status": status}
	return h.postJSON(url, body)
}

// Close closes the bridge session handle.
func (h *BridgeSessionHandle) Close() {
	h.mu.Lock()
	h.connected = false
	h.mu.Unlock()
}

// postEvent sends an event to the CCR write endpoint.
func (h *BridgeSessionHandle) postEvent(data []byte) error {
	url := fmt.Sprintf("%s/v1/code/sessions/%s/events", h.apiBaseURL, h.sessionID)
	req, err := http.NewRequest("POST", url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+h.workerJWT)
	req.Header.Set("Content-Type", "application/json")

	resp, err := h.client.Do(req)
	if err != nil {
		return fmt.Errorf("post event: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("post event: HTTP %d: %s", resp.StatusCode, string(body))
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

// AttachBridgeSessionOptions configures a bridge session attachment.
type AttachBridgeSessionOptions struct {
	// SessionID is the session ID (cse_* form).
	SessionID string
	// IngressToken is the worker JWT from fetchRemoteCredentials.
	IngressToken string
	// APIBaseURL is the session ingress API base URL.
	APIBaseURL string
	// Epoch is the worker epoch if already known. Omit to register automatically.
	Epoch *int
	// InitialSequenceNum seeds the first SSE connect's from_sequence_num.
	InitialSequenceNum int
	// HeartbeatIntervalMs is the CCRClient heartbeat interval (default 20s).
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
// Creates the transport, wires ingress routing and control dispatch,
// and returns a handle scoped to this one session.
func AttachBridgeSession(opts AttachBridgeSessionOptions) (*BridgeSessionHandle, error) {
	epoch := 0
	if opts.Epoch != nil {
		epoch = *opts.Epoch
	}

	handle := &BridgeSessionHandle{
		sessionID:            opts.SessionID,
		apiBaseURL:           opts.APIBaseURL,
		workerJWT:            opts.IngressToken,
		epoch:                epoch,
		sequenceNum:          opts.InitialSequenceNum,
		connected:            true,
		client:               &http.Client{Timeout: 30 * time.Second},
		onInboundMessage:     opts.OnInboundMessage,
		onPermissionResponse: opts.OnPermissionResponse,
		onInterrupt:          opts.OnInterrupt,
		onClose:              opts.OnClose,
	}

	return handle, nil
}

// CreateCodeSession creates a fresh CCR session via POST /v1/code/sessions.
// Returns the session ID (cse_* form), or an error.
func CreateCodeSession(baseURL, accessToken, title string, timeoutMs int, tags ...string) (string, error) {
	body := map[string]any{
		"title": title,
	}
	if len(tags) > 0 {
		body["tags"] = tags
	}

	data, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	client := &http.Client{Timeout: time.Duration(timeoutMs) * time.Millisecond}
	req, err := http.NewRequest("POST", baseURL+"/v1/code/sessions", bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
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
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("parse session response: %w", err)
	}
	return result.ID, nil
}

// FetchRemoteCredentials mints a worker JWT for a session via
// POST /v1/code/sessions/{id}/bridge.
// The call IS the worker register (bumps epoch server-side).
func FetchRemoteCredentials(sessionID, baseURL, accessToken string, timeoutMs int, trustedDeviceToken ...string) (*RemoteCredentials, error) {
	client := &http.Client{Timeout: time.Duration(timeoutMs) * time.Millisecond}
	req, err := http.NewRequest("POST", baseURL+"/v1/code/sessions/"+sessionID+"/bridge", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	if len(trustedDeviceToken) > 0 && trustedDeviceToken[0] != "" {
		req.Header.Set("X-Trusted-Device-Token", trustedDeviceToken[0])
	}

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
