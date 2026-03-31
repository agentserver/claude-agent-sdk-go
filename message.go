package agentsdk

import "encoding/json"

// SDKMessage is a tagged union of all message types from the claude process.
// Use the As* methods to access the concrete type.
type SDKMessage struct {
	Type string          `json:"type"` // "user", "assistant", "system", "result", "tool_result", "stream_event", "rate_limit", "task_started", "task_progress", "task_notification"
	Raw  json.RawMessage `json:"-"`    // Original JSON for advanced use cases
}

// UnmarshalJSON implements custom unmarshalling to capture raw JSON and extract type.
func (m *SDKMessage) UnmarshalJSON(data []byte) error {
	m.Raw = append(m.Raw[:0], data...)

	var envelope struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return err
	}
	m.Type = envelope.Type
	return nil
}

// AsUser returns the message as a UserMessage if Type == "user".
func (m SDKMessage) AsUser() (*UserMessage, bool) {
	if m.Type != "user" {
		return nil, false
	}
	var msg UserMessage
	if err := json.Unmarshal(m.Raw, &msg); err != nil {
		return nil, false
	}
	return &msg, true
}

// AsAssistant returns the message as an AssistantMessage if Type == "assistant".
func (m SDKMessage) AsAssistant() (*AssistantMessage, bool) {
	if m.Type != "assistant" {
		return nil, false
	}
	var msg AssistantMessage
	if err := json.Unmarshal(m.Raw, &msg); err != nil {
		return nil, false
	}
	return &msg, true
}

// AsSystem returns the message as a SystemMessage if Type == "system".
func (m SDKMessage) AsSystem() (*SystemMessage, bool) {
	if m.Type != "system" {
		return nil, false
	}
	var msg SystemMessage
	if err := json.Unmarshal(m.Raw, &msg); err != nil {
		return nil, false
	}
	return &msg, true
}

// AsResult returns the message as a ResultMessage if Type == "result".
func (m SDKMessage) AsResult() (*ResultMessage, bool) {
	if m.Type != "result" {
		return nil, false
	}
	var msg ResultMessage
	if err := json.Unmarshal(m.Raw, &msg); err != nil {
		return nil, false
	}
	return &msg, true
}

// AsStreamEvent returns the message as a StreamEvent if Type == "stream_event".
func (m SDKMessage) AsStreamEvent() (*StreamEvent, bool) {
	if m.Type != "stream_event" {
		return nil, false
	}
	var msg StreamEvent
	if err := json.Unmarshal(m.Raw, &msg); err != nil {
		return nil, false
	}
	return &msg, true
}

// AsRateLimit returns the message as a RateLimitEvent if Type == "rate_limit".
func (m SDKMessage) AsRateLimit() (*RateLimitEvent, bool) {
	if m.Type != "rate_limit" {
		return nil, false
	}
	var msg RateLimitEvent
	if err := json.Unmarshal(m.Raw, &msg); err != nil {
		return nil, false
	}
	return &msg, true
}

// AsToolResult returns the message as a ToolResultMessage if Type == "tool_result".
// This is a top-level message type (distinct from ToolResultBlock content block).
func (m SDKMessage) AsToolResult() (*ToolResultMessage, bool) {
	if m.Type != "tool_result" {
		return nil, false
	}
	var msg ToolResultMessage
	if err := json.Unmarshal(m.Raw, &msg); err != nil {
		return nil, false
	}
	return &msg, true
}

// AsTaskStarted returns the message as a TaskStartedMessage if Type == "task_started".
func (m SDKMessage) AsTaskStarted() (*TaskStartedMessage, bool) {
	if m.Type != "task_started" {
		return nil, false
	}
	var msg TaskStartedMessage
	if err := json.Unmarshal(m.Raw, &msg); err != nil {
		return nil, false
	}
	return &msg, true
}

// AsTaskProgress returns the message as a TaskProgressMessage if Type == "task_progress".
func (m SDKMessage) AsTaskProgress() (*TaskProgressMessage, bool) {
	if m.Type != "task_progress" {
		return nil, false
	}
	var msg TaskProgressMessage
	if err := json.Unmarshal(m.Raw, &msg); err != nil {
		return nil, false
	}
	return &msg, true
}

// AsTaskNotification returns the message as a TaskNotificationMessage if Type == "task_notification".
func (m SDKMessage) AsTaskNotification() (*TaskNotificationMessage, bool) {
	if m.Type != "task_notification" {
		return nil, false
	}
	var msg TaskNotificationMessage
	if err := json.Unmarshal(m.Raw, &msg); err != nil {
		return nil, false
	}
	return &msg, true
}

// UserMessage represents a user input message.
type UserMessage struct {
	Type            string `json:"type"`
	Content         any    `json:"content"` // string or []ContentBlock
	UUID            string `json:"uuid"`
	SessionID       string `json:"session_id"`
	ParentToolUseID string `json:"parent_tool_use_id,omitempty"`
}

// AssistantMessage represents a complete response from Claude.
type AssistantMessage struct {
	Type            string         `json:"type"`
	Content         []ContentBlock `json:"content"`
	Model           string         `json:"model"`
	StopReason      string         `json:"stop_reason"`
	UUID            string         `json:"uuid"`
	MessageID       string         `json:"message_id,omitempty"`
	SessionID       string         `json:"session_id"`
	ParentToolUseID string         `json:"parent_tool_use_id,omitempty"`
	Usage           *MessageUsage  `json:"usage,omitempty"`
	// Error is set when the assistant response encountered an error.
	// Values: "authentication_failed", "billing_error", "rate_limit",
	// "invalid_request", "server_error", "max_output_tokens", "unknown".
	Error string `json:"error,omitempty"`
}

// SystemMessage represents a system event (init, status, etc.).
type SystemMessage struct {
	Type      string          `json:"type"`
	Subtype   string          `json:"subtype"` // "init", "api_retry", "mcp_status", etc.
	UUID      string          `json:"uuid"`
	SessionID string          `json:"session_id"`
	Data      json.RawMessage `json:"data,omitempty"` // Subtype-specific payload (MCP status, init data, etc.)
}

// ResultMessage is the final message with execution summary.
// Subtype is one of: "success", "error_max_turns", "error_during_execution",
// "error_max_budget_usd", "error_max_structured_output_retries", "paused".
type ResultMessage struct {
	Type              string             `json:"type"`
	Subtype           string             `json:"subtype"`
	SessionID         string             `json:"session_id"`
	DurationMs        int                `json:"duration_ms"`
	DurationAPIMs     int                `json:"duration_api_ms"`
	IsError           bool               `json:"is_error"`
	NumTurns          int                `json:"num_turns"`
	TotalCostUSD      *float64           `json:"total_cost_usd,omitempty"`
	Result            string             `json:"result"`
	Errors            []string           `json:"errors,omitempty"`            // Error details when subtype is error_*
	StopReason        string             `json:"stop_reason"`
	Usage             *ResultUsage       `json:"usage,omitempty"`
	ModelUsage        map[string]any     `json:"model_usage,omitempty"`       // Per-model usage breakdown
	StructuredOutput  any                `json:"structured_output,omitempty"` // Structured output when output_format is set
	PermissionDenials []PermissionDenial `json:"permission_denials,omitempty"`
}

// PermissionDenial records a tool use that was denied by permission policy.
type PermissionDenial struct {
	ToolName  string         `json:"tool_name"`
	ToolUseID string         `json:"tool_use_id"`
	ToolInput map[string]any `json:"tool_input"`
}

// StreamEvent represents a partial streaming event (token deltas, etc.).
type StreamEvent struct {
	Type      string          `json:"type"`
	Event     json.RawMessage `json:"event"` // Varies by event type
	SessionID string          `json:"session_id"`
	UUID      string          `json:"uuid"`
}

// RateLimitEvent is emitted when a rate limit is encountered.
type RateLimitEvent struct {
	Type          string         `json:"type"`
	RateLimitInfo *RateLimitInfo `json:"rate_limit_info"`
	SessionID     string         `json:"session_id"`
	UUID          string         `json:"uuid"`
}

// RateLimitInfo contains details about a rate limit.
type RateLimitInfo struct {
	Status                string  `json:"status"`                            // "allowed", "allowed_warning", "rejected"
	ResetsAt              string  `json:"resets_at"`                         // ISO 8601 timestamp
	RateLimitType         string  `json:"rate_limit_type"`                   // "five_hour", "seven_day", "seven_day_opus", "seven_day_sonnet", "overage"
	Utilization           float64 `json:"utilization"`                       // 0.0–1.0
	RequestsRemaining     *int    `json:"requests_remaining,omitempty"`      // Remaining requests in window
	RequestsLimit         *int    `json:"requests_limit,omitempty"`          // Total request limit
	TokensRemaining       *int    `json:"tokens_remaining,omitempty"`        // Remaining tokens in window
	TokensLimit           *int    `json:"tokens_limit,omitempty"`            // Total token limit
	OverageStatus         string  `json:"overage_status,omitempty"`          // Overage billing status
	OverageResetsAt       string  `json:"overage_resets_at,omitempty"`       // Overage reset timestamp
	OverageDisabledReason string  `json:"overage_disabled_reason,omitempty"` // Why overage is disabled
}

// ToolResultMessage represents the result of a tool execution.
type ToolResultMessage struct {
	Type            string `json:"type"`
	ToolUseID       string `json:"tool_use_id"`
	Content         any    `json:"content"` // string or structured content
	IsError         *bool  `json:"is_error,omitempty"`
	SessionID       string `json:"session_id"`
	UUID            string `json:"uuid"`
	ParentToolUseID string `json:"parent_tool_use_id,omitempty"`
}

// TaskStartedMessage is emitted when a background task begins execution.
type TaskStartedMessage struct {
	Type      string `json:"type"`
	TaskID    string `json:"task_id"`
	SessionID string `json:"session_id"`
	UUID      string `json:"uuid"`
}

// TaskProgressMessage is emitted periodically while a background task runs.
type TaskProgressMessage struct {
	Type      string     `json:"type"`
	TaskID    string     `json:"task_id"`
	SessionID string     `json:"session_id"`
	UUID      string     `json:"uuid"`
	Usage     *TaskUsage `json:"usage,omitempty"`
}

// TaskNotificationMessage is emitted when a background task completes, fails, or is stopped.
type TaskNotificationMessage struct {
	Type       string     `json:"type"`
	TaskID     string     `json:"task_id"`
	Status     string     `json:"status"` // "completed", "failed", "stopped"
	OutputFile string     `json:"output_file,omitempty"`
	Summary    string     `json:"summary,omitempty"`
	SessionID  string     `json:"session_id"`
	UUID       string     `json:"uuid"`
	Usage      *TaskUsage `json:"usage,omitempty"`
}

// TaskUsage tracks resource usage for a background task.
type TaskUsage struct {
	TotalTokens int   `json:"total_tokens"`
	ToolUses    int   `json:"tool_uses"`
	DurationMs  int64 `json:"duration_ms"`
}

// MessageUsage tracks token usage for a single message.
type MessageUsage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens,omitempty"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens,omitempty"`
}

// ResultUsage tracks aggregate token usage for the entire session.
type ResultUsage struct {
	InputTokens              int      `json:"input_tokens"`
	OutputTokens             int      `json:"output_tokens"`
	CacheCreationInputTokens int      `json:"cache_creation_input_tokens,omitempty"`
	CacheReadInputTokens     int      `json:"cache_read_input_tokens,omitempty"`
	CostUSD                  *float64 `json:"cost_usd,omitempty"`
}
