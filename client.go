package agentsdk

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/anthropics/claude-agent-sdk-go/internal/clilookup"
	"github.com/anthropics/claude-agent-sdk-go/internal/transport"
)

// Client maintains a persistent connection to a claude subprocess
// for multi-turn conversations.
type Client struct {
	config         queryConfig
	transport      transport.Transport
	controlHandler *controlHandler
	msgCh          chan SDKMessage
	cancel         context.CancelFunc
	ctx            context.Context
	closed         atomic.Bool
	mu             sync.Mutex
	reqID          atomic.Int64

	// Control protocol response routing.
	pendingMu sync.Mutex
	pending   map[string]chan json.RawMessage // request_id → response channel
}

// NewClient creates a new interactive Client with the given options.
func NewClient(opts ...QueryOption) *Client {
	cfg := queryConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}
	cfg.applyDefaults()

	return &Client{
		config:  cfg,
		msgCh:   make(chan SDKMessage, 64),
		pending: make(map[string]chan json.RawMessage),
	}
}

// Connect starts the claude subprocess and begins reading messages.
func (c *Client) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	cliPath, err := clilookup.FindCLI(c.config.cliPath)
	if err != nil {
		return err
	}

	args := buildCLIArgs(&c.config)

	// Merge API key into env if set.
	env := c.config.env
	if c.config.apiKey != "" {
		if env == nil {
			env = make(map[string]string)
		}
		env["ANTHROPIC_API_KEY"] = c.config.apiKey
	}

	tp := transport.NewSubprocessTransport(transport.SubprocessConfig{
		CLIPath:       cliPath,
		Args:          args,
		Cwd:           c.config.cwd,
		Env:           env,
		MaxBufferSize: c.config.maxBufferSize,
		StderrFunc:    c.config.stderrFunc,
	})

	ctx, cancel := context.WithCancel(ctx)
	c.cancel = cancel

	if err := tp.Connect(ctx); err != nil {
		cancel()
		return err
	}

	c.transport = tp
	c.ctx = ctx

	// Create control handler for hooks, permissions, and MCP tool calls.
	c.controlHandler = newControlHandler(&c.config, tp)

	// Send initialize handshake to register hooks and agents with the CLI.
	if err := c.controlHandler.sendInitialize(ctx); err != nil {
		cancel()
		tp.Close()
		return fmt.Errorf("initialize: %w", err)
	}

	// Read messages from transport, route control requests/responses, forward others.
	go func() {
		for raw := range tp.ReadMessages() {
			// Route control requests (hook callbacks, permission checks, MCP tool calls).
			if c.controlHandler.handleMessage(ctx, raw) {
				continue
			}

			// Check if this is a control response (for Client-initiated requests).
			var envelope struct {
				Type      string `json:"type"`
				RequestID string `json:"request_id"`
			}
			if json.Unmarshal(raw, &envelope) == nil && envelope.Type == "control_response" {
				c.pendingMu.Lock()
				if ch, ok := c.pending[envelope.RequestID]; ok {
					ch <- raw
					delete(c.pending, envelope.RequestID)
				}
				c.pendingMu.Unlock()
				continue
			}

			var msg SDKMessage
			if err := json.Unmarshal(raw, &msg); err != nil {
				continue
			}
			c.msgCh <- msg
		}
		close(c.msgCh)
	}()

	return nil
}

// Send sends a user message to the claude process.
func (c *Client) Send(ctx context.Context, prompt string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.transport == nil || !c.transport.IsReady() {
		return fmt.Errorf("client not connected")
	}

	userMsg := map[string]any{
		"type": "user",
		"message": map[string]any{
			"role":    "user",
			"content": prompt,
		},
	}
	data, err := json.Marshal(userMsg)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}
	return c.transport.Write(string(data))
}

// Messages returns a channel that yields SDKMessages as they arrive.
func (c *Client) Messages() <-chan SDKMessage {
	return c.msgCh
}

// Interrupt sends an interrupt signal to the claude process.
func (c *Client) Interrupt() error {
	return c.sendControlRequest("interrupt", nil)
}

// SetModel changes the model at runtime.
func (c *Client) SetModel(model string) error {
	return c.sendControlRequest("set_model", map[string]any{"model": model})
}

// SetPermissionMode changes the permission mode at runtime.
func (c *Client) SetPermissionMode(mode PermissionMode) error {
	return c.sendControlRequest("set_permission_mode", map[string]any{"mode": string(mode)})
}

// SetMaxThinkingTokens changes the max thinking tokens at runtime.
func (c *Client) SetMaxThinkingTokens(tokens int) error {
	return c.sendControlRequest("set_max_thinking_tokens", map[string]any{"max_thinking_tokens": tokens})
}

// ContextUsage contains context window usage information.
type ContextUsage struct {
	TotalTokens          int               `json:"totalTokens"`
	MaxTokens            int               `json:"maxTokens"`
	Percentage           float64           `json:"percentage"` // 0–100
	Model                string            `json:"model"`
	IsAutoCompactEnabled bool              `json:"isAutoCompactEnabled"`
	Categories           []ContextCategory `json:"categories,omitempty"`
	McpTools             []string          `json:"mcpTools,omitempty"`
	Agents               []string          `json:"agents,omitempty"`
	MemoryFiles          []string          `json:"memoryFiles,omitempty"`
}

// ContextCategory is a single category in the context usage breakdown.
type ContextCategory struct {
	Name   string `json:"name"`
	Tokens int    `json:"tokens"`
	Color  string `json:"color"`
}

// GetContextUsage returns context window usage information.
func (c *Client) GetContextUsage(ctx context.Context) (*ContextUsage, error) {
	raw, err := c.sendControlRequestWithResponse(ctx, "get_context_usage", nil)
	if err != nil {
		return nil, err
	}
	var usage ContextUsage
	if err := json.Unmarshal(raw, &usage); err != nil {
		return nil, fmt.Errorf("parse context usage: %w", err)
	}
	return &usage, nil
}

// McpStatus requests the status of all MCP servers.
func (c *Client) McpStatus(ctx context.Context) ([]McpServerStatus, error) {
	raw, err := c.sendControlRequestWithResponse(ctx, "mcp_status", nil)
	if err != nil {
		return nil, err
	}
	var resp struct {
		McpServers []McpServerStatus `json:"mcpServers"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("parse mcp status: %w", err)
	}
	return resp.McpServers, nil
}

// ReconnectMcpServer reconnects a failed MCP server by name.
func (c *Client) ReconnectMcpServer(serverName string) error {
	return c.sendControlRequest("reconnect_mcp_server", map[string]any{"server_name": serverName})
}

// ToggleMcpServer enables or disables an MCP server at runtime.
func (c *Client) ToggleMcpServer(serverName string, enabled bool) error {
	return c.sendControlRequest("toggle_mcp_server", map[string]any{
		"server_name": serverName,
		"enabled":     enabled,
	})
}

// SetMcpServers dynamically replaces all MCP server configurations.
func (c *Client) SetMcpServers(servers map[string]McpServerConfig) error {
	serversJSON, err := json.Marshal(servers)
	if err != nil {
		return fmt.Errorf("marshal servers: %w", err)
	}
	var serversMap map[string]any
	json.Unmarshal(serversJSON, &serversMap)
	return c.sendControlRequest("set_mcp_servers", map[string]any{"servers": serversMap})
}

// StopTask stops a background task.
func (c *Client) StopTask(taskID string) error {
	return c.sendControlRequest("stop_task", map[string]any{"task_id": taskID})
}

// RewindFiles reverts file changes to the state at a given user message.
// Requires WithFileCheckpointing to be enabled.
func (c *Client) RewindFiles(userMessageID string) error {
	return c.sendControlRequest("rewind_files", map[string]any{"user_message_id": userMessageID})
}

// ServerInfo contains information about the claude server's capabilities.
type ServerInfo struct {
	Commands    []string       `json:"commands,omitempty"`
	OutputStyle string         `json:"outputStyle,omitempty"`
	Extra       map[string]any `json:"-"`
}

// GetServerInfo returns information about the claude server's capabilities.
func (c *Client) GetServerInfo(ctx context.Context) (*ServerInfo, error) {
	raw, err := c.sendControlRequestWithResponse(ctx, "get_server_info", nil)
	if err != nil {
		return nil, err
	}
	var info ServerInfo
	if err := json.Unmarshal(raw, &info); err != nil {
		return nil, fmt.Errorf("parse server info: %w", err)
	}
	return &info, nil
}

// PromptSuggestion requests prompt suggestions based on the current context.
func (c *Client) PromptSuggestion(ctx context.Context) ([]string, error) {
	raw, err := c.sendControlRequestWithResponse(ctx, "prompt_suggestion", nil)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Suggestions []string `json:"suggestions"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("parse prompt suggestions: %w", err)
	}
	return resp.Suggestions, nil
}

// EnableChannel activates an MCP server channel by name.
func (c *Client) EnableChannel(serverName, channel string) error {
	return c.sendControlRequest("enable_channel", map[string]any{
		"server_name": serverName,
		"channel":     channel,
	})
}

// RuntimeSettings contains the resolved runtime settings from the claude process.
type RuntimeSettings struct {
	Model  string         `json:"model"`
	Effort string         `json:"effort"`
	Extra  map[string]any `json:"-"`
}

// GetSettings returns the applied runtime settings with resolved model and effort values.
func (c *Client) GetSettings(ctx context.Context) (*RuntimeSettings, error) {
	raw, err := c.sendControlRequestWithResponse(ctx, "get_settings", nil)
	if err != nil {
		return nil, err
	}
	var settings RuntimeSettings
	if err := json.Unmarshal(raw, &settings); err != nil {
		return nil, fmt.Errorf("parse settings: %w", err)
	}
	return &settings, nil
}

// ReloadPluginsResult contains refreshed plugin state after reloading.
type ReloadPluginsResult struct {
	Commands   []string          `json:"commands,omitempty"`
	Agents     map[string]any    `json:"agents,omitempty"`
	McpServers []McpServerStatus `json:"mcpServers,omitempty"`
}

// ReloadPlugins reloads all plugins and returns refreshed commands, agents, and MCP servers.
func (c *Client) ReloadPlugins(ctx context.Context) (*ReloadPluginsResult, error) {
	raw, err := c.sendControlRequestWithResponse(ctx, "reload_plugins", nil)
	if err != nil {
		return nil, err
	}
	var result ReloadPluginsResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("parse reload plugins: %w", err)
	}
	return &result, nil
}

// Close shuts down the claude subprocess and releases resources.
func (c *Client) Close() error {
	if c.closed.Swap(true) {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cancel != nil {
		c.cancel()
	}
	if c.transport != nil {
		return c.transport.Close()
	}
	return nil
}

// sendControlRequest sends a fire-and-forget control request to the claude process.
func (c *Client) sendControlRequest(requestType string, params map[string]any) error {
	_, err := c.writeControlRequest(requestType, params)
	return err
}

// sendControlRequestWithResponse sends a control request and waits for the response.
func (c *Client) sendControlRequestWithResponse(ctx context.Context, requestType string, params map[string]any) (json.RawMessage, error) {
	reqID, err := c.writeControlRequest(requestType, params)
	if err != nil {
		return nil, err
	}

	// Register response channel.
	ch := make(chan json.RawMessage, 1)
	c.pendingMu.Lock()
	c.pending[reqID] = ch
	c.pendingMu.Unlock()

	// Wait for response or context cancellation.
	select {
	case raw := <-ch:
		return raw, nil
	case <-ctx.Done():
		c.pendingMu.Lock()
		delete(c.pending, reqID)
		c.pendingMu.Unlock()
		return nil, ctx.Err()
	}
}

// writeControlRequest marshals and writes a control request, returning its ID.
func (c *Client) writeControlRequest(requestType string, params map[string]any) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.transport == nil || !c.transport.IsReady() {
		return "", fmt.Errorf("client not connected")
	}

	reqID := fmt.Sprintf("sdk_%d", c.reqID.Add(1))
	request := map[string]any{"subtype": requestType}
	for k, v := range params {
		request[k] = v
	}

	msg := map[string]any{
		"type":       "control_request",
		"request_id": reqID,
		"request":    request,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return "", err
	}
	return reqID, c.transport.Write(string(data))
}

// --- V2 Preview API (unstable — may change) ---

// V2Session represents a session handle returned by V2 API methods.
type V2Session struct {
	ID     string `json:"id"`
	client *Client
}

// UnstableV2CreateSession creates a new session and returns a session handle.
func UnstableV2CreateSession(ctx context.Context, opts ...QueryOption) (*V2Session, error) {
	client := NewClient(opts...)
	if err := client.Connect(ctx); err != nil {
		return nil, err
	}

	// Wait for init message to get session ID.
	for msg := range client.Messages() {
		if sys, ok := msg.AsSystem(); ok && sys.Subtype == "init" {
			return &V2Session{
				ID:     sys.SessionID,
				client: client,
			}, nil
		}
	}
	return nil, fmt.Errorf("no init message received")
}

// UnstableV2ResumeSession resumes an existing session by ID.
func UnstableV2ResumeSession(ctx context.Context, sessionID string, opts ...QueryOption) (*V2Session, error) {
	opts = append(opts, WithResume(sessionID))
	client := NewClient(opts...)
	if err := client.Connect(ctx); err != nil {
		return nil, err
	}
	return &V2Session{
		ID:     sessionID,
		client: client,
	}, nil
}

// Send sends a prompt to the session and returns a channel of response messages.
func (s *V2Session) Send(ctx context.Context, prompt string) (<-chan SDKMessage, error) {
	if err := s.client.Send(ctx, prompt); err != nil {
		return nil, err
	}
	return s.client.Messages(), nil
}

// Close closes the V2 session.
func (s *V2Session) Close() error {
	return s.client.Close()
}

// UnstableV2Prompt is a one-shot convenience that creates a session, sends
// one message, waits for the result, and returns the final ResultMessage.
func UnstableV2Prompt(ctx context.Context, prompt string, opts ...QueryOption) (*ResultMessage, error) {
	stream := Query(ctx, prompt, opts...)
	defer stream.Close()
	return stream.Result()
}

// --- Client inspection methods ---

// SlashCommand describes an available slash command.
type SlashCommand struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	ArgumentHint string `json:"argumentHint,omitempty"`
}

// SupportedCommands returns the list of available slash commands.
func (c *Client) SupportedCommands(ctx context.Context) ([]SlashCommand, error) {
	raw, err := c.sendControlRequestWithResponse(ctx, "supported_commands", nil)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Commands []SlashCommand `json:"commands"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("parse supported commands: %w", err)
	}
	return resp.Commands, nil
}

// ModelInfo describes an available model.
type ModelInfo struct {
	DisplayName              string   `json:"displayName"`
	Description              string   `json:"description,omitempty"`
	SupportsEffort           bool     `json:"supportsEffort"`
	SupportedEffortLevels    []string `json:"supportedEffortLevels,omitempty"`
	SupportsAdaptiveThinking bool     `json:"supportsAdaptiveThinking"`
	SupportsFastMode         bool     `json:"supportsFastMode"`
}

// SupportedModels returns the list of available models with metadata.
func (c *Client) SupportedModels(ctx context.Context) ([]ModelInfo, error) {
	raw, err := c.sendControlRequestWithResponse(ctx, "supported_models", nil)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Models []ModelInfo `json:"models"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("parse supported models: %w", err)
	}
	return resp.Models, nil
}

// AgentInfo describes an available agent.
type AgentInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Model       string `json:"model,omitempty"`
}

// SupportedAgents returns the list of available agents.
func (c *Client) SupportedAgents(ctx context.Context) ([]AgentInfo, error) {
	raw, err := c.sendControlRequestWithResponse(ctx, "supported_agents", nil)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Agents []AgentInfo `json:"agents"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("parse supported agents: %w", err)
	}
	return resp.Agents, nil
}

// AccountInfo contains account-level information.
type AccountInfo struct {
	OrganizationID   string `json:"organizationId,omitempty"`
	OrganizationName string `json:"organizationName,omitempty"`
	AccountUUID      string `json:"accountUuid,omitempty"`
	EmailAddress     string `json:"emailAddress,omitempty"`
}

// GetAccountInfo returns account-level information.
func (c *Client) GetAccountInfo(ctx context.Context) (*AccountInfo, error) {
	raw, err := c.sendControlRequestWithResponse(ctx, "account_info", nil)
	if err != nil {
		return nil, err
	}
	var info AccountInfo
	if err := json.Unmarshal(raw, &info); err != nil {
		return nil, fmt.Errorf("parse account info: %w", err)
	}
	return &info, nil
}

// SeedReadState seeds the file read state for change tracking.
func (c *Client) SeedReadState(path string, mtime int64) error {
	return c.sendControlRequest("seed_read_state", map[string]any{
		"path":  path,
		"mtime": mtime,
	})
}

// ApplyFlagSettings applies runtime settings programmatically.
func (c *Client) ApplyFlagSettings(settings map[string]any) error {
	return c.sendControlRequest("apply_flag_settings", map[string]any{
		"settings": settings,
	})
}

// FastModeState represents the fast mode toggle state.
type FastModeState string

const (
	FastModeOff      FastModeState = "off"
	FastModeOn       FastModeState = "on"
	FastModeCooldown FastModeState = "cooldown"
)

// SetFastMode toggles fast mode on or off.
func (c *Client) SetFastMode(enabled bool) error {
	return c.sendControlRequest("set_fast_mode", map[string]any{
		"enabled": enabled,
	})
}
