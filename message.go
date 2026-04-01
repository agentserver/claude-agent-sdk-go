package agentsdk

import "encoding/json"

// SDKMessage is a tagged union of all message types from the claude process.
// Use the As* methods to access the concrete type.
//
// Supported types:
//   - "user"              → AsUser() or AsUserReplay() (if isReplay is true)
//   - "assistant"         → AsAssistant()
//   - "system"            → AsSystem() (generic), or specific subtypes:
//   - "system/init"       → AsSystem()
//   - "system/api_retry"  → AsAPIRetry()
//   - "system/compact_boundary"       → AsCompactBoundary()
//   - "system/hook_started"           → AsHookStarted()
//   - "system/hook_progress"          → AsHookProgress()
//   - "system/hook_response"          → AsHookResponse()
//   - "system/local_command_output"   → AsLocalCommandOutput()
//   - "system/files_persisted"        → AsFilesPersisted()
//   - "system/elicitation_complete"   → AsElicitationComplete()
//   - "system/session_state_changed"  → AsSessionStateChanged()
//   - "system/status"                 → AsStatus()
//   - "system/task_started"           → AsTaskStarted()
//   - "system/task_progress"          → AsTaskProgress()
//   - "system/task_notification"      → AsTaskNotification()
//   - "result"            → AsResult()
//   - "tool_result"       → AsToolResult()
//   - "stream_event"      → AsStreamEvent()
//   - "rate_limit_event"  → AsRateLimit()
//   - "auth_status"       → AsAuthStatus()
//   - "tool_progress"     → AsToolProgress()
//   - "tool_use_summary"  → AsToolUseSummary()
//   - "prompt_suggestion" → AsPromptSuggestion()
type SDKMessage struct {
	Type    string          `json:"type"`
	Subtype string          `json:"-"` // Populated for "system" type messages
	Raw     json.RawMessage `json:"-"` // Original JSON for advanced use cases
}

// UnmarshalJSON implements custom unmarshalling to capture raw JSON and extract type/subtype.
func (m *SDKMessage) UnmarshalJSON(data []byte) error {
	m.Raw = append(m.Raw[:0], data...)

	var envelope struct {
		Type    string `json:"type"`
		Subtype string `json:"subtype"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return err
	}
	m.Type = envelope.Type
	m.Subtype = envelope.Subtype
	return nil
}

// --- Core message type accessors ---

// AsUser returns the message as a UserMessage if Type == "user".
// This returns both regular and replayed user messages. Use AsUserReplay()
// to specifically check for replayed messages.
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
// For specific system subtypes, use the dedicated AsXxx() methods.
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

// AsRateLimit returns the message as a RateLimitEvent if Type == "rate_limit_event".
func (m SDKMessage) AsRateLimit() (*RateLimitEvent, bool) {
	if m.Type != "rate_limit_event" {
		return nil, false
	}
	var msg RateLimitEvent
	if err := json.Unmarshal(m.Raw, &msg); err != nil {
		return nil, false
	}
	return &msg, true
}

// AsToolResult returns the message as a ToolResultMessage if Type == "tool_result".
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

// --- System subtype accessors (type == "system") ---

// AsTaskStarted returns the message if type == "system" and subtype == "task_started".
func (m SDKMessage) AsTaskStarted() (*TaskStartedMessage, bool) {
	if m.Type != "system" || m.Subtype != "task_started" {
		return nil, false
	}
	var msg TaskStartedMessage
	if err := json.Unmarshal(m.Raw, &msg); err != nil {
		return nil, false
	}
	return &msg, true
}

// AsTaskProgress returns the message if type == "system" and subtype == "task_progress".
func (m SDKMessage) AsTaskProgress() (*TaskProgressMessage, bool) {
	if m.Type != "system" || m.Subtype != "task_progress" {
		return nil, false
	}
	var msg TaskProgressMessage
	if err := json.Unmarshal(m.Raw, &msg); err != nil {
		return nil, false
	}
	return &msg, true
}

// AsTaskNotification returns the message if type == "system" and subtype == "task_notification".
func (m SDKMessage) AsTaskNotification() (*TaskNotificationMessage, bool) {
	if m.Type != "system" || m.Subtype != "task_notification" {
		return nil, false
	}
	var msg TaskNotificationMessage
	if err := json.Unmarshal(m.Raw, &msg); err != nil {
		return nil, false
	}
	return &msg, true
}

// AsAPIRetry returns the message if type == "system" and subtype == "api_retry".
func (m SDKMessage) AsAPIRetry() (*APIRetryMessage, bool) {
	if m.Type != "system" || m.Subtype != "api_retry" {
		return nil, false
	}
	var msg APIRetryMessage
	if err := json.Unmarshal(m.Raw, &msg); err != nil {
		return nil, false
	}
	return &msg, true
}

// AsCompactBoundary returns the message if type == "system" and subtype == "compact_boundary".
func (m SDKMessage) AsCompactBoundary() (*CompactBoundaryMessage, bool) {
	if m.Type != "system" || m.Subtype != "compact_boundary" {
		return nil, false
	}
	var msg CompactBoundaryMessage
	if err := json.Unmarshal(m.Raw, &msg); err != nil {
		return nil, false
	}
	return &msg, true
}

// AsHookStarted returns the message if type == "system" and subtype == "hook_started".
func (m SDKMessage) AsHookStarted() (*HookStartedMessage, bool) {
	if m.Type != "system" || m.Subtype != "hook_started" {
		return nil, false
	}
	var msg HookStartedMessage
	if err := json.Unmarshal(m.Raw, &msg); err != nil {
		return nil, false
	}
	return &msg, true
}

// AsHookProgress returns the message if type == "system" and subtype == "hook_progress".
func (m SDKMessage) AsHookProgress() (*HookProgressMessage, bool) {
	if m.Type != "system" || m.Subtype != "hook_progress" {
		return nil, false
	}
	var msg HookProgressMessage
	if err := json.Unmarshal(m.Raw, &msg); err != nil {
		return nil, false
	}
	return &msg, true
}

// AsHookResponse returns the message if type == "system" and subtype == "hook_response".
func (m SDKMessage) AsHookResponse() (*HookResponseMessage, bool) {
	if m.Type != "system" || m.Subtype != "hook_response" {
		return nil, false
	}
	var msg HookResponseMessage
	if err := json.Unmarshal(m.Raw, &msg); err != nil {
		return nil, false
	}
	return &msg, true
}

// AsLocalCommandOutput returns the message if type == "system" and subtype == "local_command_output".
func (m SDKMessage) AsLocalCommandOutput() (*LocalCommandOutputMessage, bool) {
	if m.Type != "system" || m.Subtype != "local_command_output" {
		return nil, false
	}
	var msg LocalCommandOutputMessage
	if err := json.Unmarshal(m.Raw, &msg); err != nil {
		return nil, false
	}
	return &msg, true
}

// AsFilesPersisted returns the message if type == "system" and subtype == "files_persisted".
func (m SDKMessage) AsFilesPersisted() (*FilesPersistedEvent, bool) {
	if m.Type != "system" || m.Subtype != "files_persisted" {
		return nil, false
	}
	var msg FilesPersistedEvent
	if err := json.Unmarshal(m.Raw, &msg); err != nil {
		return nil, false
	}
	return &msg, true
}

// AsElicitationComplete returns the message if type == "system" and subtype == "elicitation_complete".
func (m SDKMessage) AsElicitationComplete() (*ElicitationCompleteMessage, bool) {
	if m.Type != "system" || m.Subtype != "elicitation_complete" {
		return nil, false
	}
	var msg ElicitationCompleteMessage
	if err := json.Unmarshal(m.Raw, &msg); err != nil {
		return nil, false
	}
	return &msg, true
}

// AsSessionStateChanged returns the message if type == "system" and subtype == "session_state_changed".
func (m SDKMessage) AsSessionStateChanged() (*SessionStateChangedMessage, bool) {
	if m.Type != "system" || m.Subtype != "session_state_changed" {
		return nil, false
	}
	var msg SessionStateChangedMessage
	if err := json.Unmarshal(m.Raw, &msg); err != nil {
		return nil, false
	}
	return &msg, true
}

// AsStatus returns the message if type == "system" and subtype == "status".
func (m SDKMessage) AsStatus() (*StatusMessage, bool) {
	if m.Type != "system" || m.Subtype != "status" {
		return nil, false
	}
	var msg StatusMessage
	if err := json.Unmarshal(m.Raw, &msg); err != nil {
		return nil, false
	}
	return &msg, true
}

// AsInit returns the message if type == "system" and subtype == "init".
func (m SDKMessage) AsInit() (*InitMessage, bool) {
	if m.Type != "system" || m.Subtype != "init" {
		return nil, false
	}
	var msg InitMessage
	if err := json.Unmarshal(m.Raw, &msg); err != nil {
		return nil, false
	}
	return &msg, true
}

// --- Other top-level type accessors ---

// AsUserReplay returns the message as a UserMessageReplay if Type == "user" and IsReplay is true.
// UserMessageReplay shares the same wire type ("user") as UserMessage,
// distinguished by the isReplay field.
func (m SDKMessage) AsUserReplay() (*UserMessageReplay, bool) {
	if m.Type != "user" {
		return nil, false
	}
	var msg UserMessageReplay
	if err := json.Unmarshal(m.Raw, &msg); err != nil {
		return nil, false
	}
	if !msg.IsReplay {
		return nil, false
	}
	return &msg, true
}

// AsAuthStatus returns the message as an AuthStatusMessage if Type == "auth_status".
func (m SDKMessage) AsAuthStatus() (*AuthStatusMessage, bool) {
	if m.Type != "auth_status" {
		return nil, false
	}
	var msg AuthStatusMessage
	if err := json.Unmarshal(m.Raw, &msg); err != nil {
		return nil, false
	}
	return &msg, true
}

// AsToolProgress returns the message as a ToolProgressMessage if Type == "tool_progress".
func (m SDKMessage) AsToolProgress() (*ToolProgressMessage, bool) {
	if m.Type != "tool_progress" {
		return nil, false
	}
	var msg ToolProgressMessage
	if err := json.Unmarshal(m.Raw, &msg); err != nil {
		return nil, false
	}
	return &msg, true
}

// AsToolUseSummary returns the message as a ToolUseSummaryMessage if Type == "tool_use_summary".
func (m SDKMessage) AsToolUseSummary() (*ToolUseSummaryMessage, bool) {
	if m.Type != "tool_use_summary" {
		return nil, false
	}
	var msg ToolUseSummaryMessage
	if err := json.Unmarshal(m.Raw, &msg); err != nil {
		return nil, false
	}
	return &msg, true
}

// AsPromptSuggestion returns the message as a PromptSuggestionMessage if Type == "prompt_suggestion".
func (m SDKMessage) AsPromptSuggestion() (*PromptSuggestionMessage, bool) {
	if m.Type != "prompt_suggestion" {
		return nil, false
	}
	var msg PromptSuggestionMessage
	if err := json.Unmarshal(m.Raw, &msg); err != nil {
		return nil, false
	}
	return &msg, true
}

// =============================================================================
// Core message types
// =============================================================================

// APIMessage represents the Anthropic API message payload within an AssistantMessage.
// This corresponds to the TS SDK's BetaMessage type.
type APIMessage struct {
	ID         string         `json:"id,omitempty"`
	Type       string         `json:"type,omitempty"` // "message"
	Role       string         `json:"role,omitempty"` // "assistant"
	Content    []ContentBlock `json:"content"`
	Model      string         `json:"model,omitempty"`
	StopReason string         `json:"stop_reason,omitempty"`
	Usage      *MessageUsage  `json:"usage,omitempty"`
}

// AssistantMessage represents a complete response from Claude.
// The API message (content, model, etc.) is nested under the Message field,
// matching the wire format defined by the TS SDK.
type AssistantMessage struct {
	Type            string     `json:"type"`
	Message         APIMessage `json:"message"`
	ParentToolUseID *string    `json:"parent_tool_use_id"`
	// Error is set when the assistant response encountered an error.
	// Values: "authentication_failed", "billing_error", "rate_limit",
	// "invalid_request", "server_error", "max_output_tokens", "unknown".
	Error     string `json:"error,omitempty"`
	UUID      string `json:"uuid"`
	SessionID string `json:"session_id"`
}

// UserAPIMessage represents the Anthropic API message payload within a UserMessage.
type UserAPIMessage struct {
	Role    string `json:"role"`    // "user"
	Content any    `json:"content"` // string or []ContentBlock
}

// UserMessage represents a user input message.
type UserMessage struct {
	Type            string         `json:"type"`
	Message         UserAPIMessage `json:"message"`
	ParentToolUseID *string        `json:"parent_tool_use_id"`
	IsSynthetic     bool           `json:"isSynthetic,omitempty"`
	IsReplay        bool           `json:"isReplay,omitempty"`
	ToolUseResult   any            `json:"tool_use_result,omitempty"`
	Priority        string         `json:"priority,omitempty"` // "now", "next", "later"
	Timestamp       string         `json:"timestamp,omitempty"`
	UUID            string         `json:"uuid,omitempty"`
	SessionID       string         `json:"session_id,omitempty"`
}

// SystemMessage represents a system event (init, status, etc.).
// For specific subtypes, use the dedicated AsXxx() methods on SDKMessage.
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
	UUID              string             `json:"uuid,omitempty"`
}

// PermissionDenial records a tool use that was denied by permission policy.
type PermissionDenial struct {
	ToolName  string         `json:"tool_name"`
	ToolUseID string         `json:"tool_use_id"`
	ToolInput map[string]any `json:"tool_input"`
}

// StreamEvent represents a partial streaming event (token deltas, etc.).
type StreamEvent struct {
	Type            string          `json:"type"`
	Event           json.RawMessage `json:"event"` // Varies by event type
	ParentToolUseID *string         `json:"parent_tool_use_id"`
	SessionID       string          `json:"session_id"`
	UUID            string          `json:"uuid"`
}

// RateLimitEvent is emitted when a rate limit is encountered.
type RateLimitEvent struct {
	Type          string         `json:"type"` // "rate_limit_event"
	RateLimitInfo *RateLimitInfo `json:"rate_limit_info"`
	SessionID     string         `json:"session_id"`
	UUID          string         `json:"uuid"`
}

// RateLimitInfo contains details about a rate limit.
// Field names use camelCase to match the TS SDK wire format.
type RateLimitInfo struct {
	Status                string   `json:"status"`                          // "allowed", "allowed_warning", "rejected"
	ResetsAt              *float64 `json:"resetsAt,omitempty"`              // Epoch milliseconds
	RateLimitType         string   `json:"rateLimitType,omitempty"`         // "five_hour", "seven_day", "seven_day_opus", "seven_day_sonnet", "overage"
	Utilization           float64  `json:"utilization,omitempty"`           // 0.0–1.0
	OverageStatus         string   `json:"overageStatus,omitempty"`         // "allowed", "allowed_warning", "rejected"
	OverageResetsAt       *float64 `json:"overageResetsAt,omitempty"`       // Epoch milliseconds
	OverageDisabledReason string   `json:"overageDisabledReason,omitempty"` // Why overage is disabled
	IsUsingOverage        bool     `json:"isUsingOverage,omitempty"`
	SurpassedThreshold    *float64 `json:"surpassedThreshold,omitempty"`
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
// Wire format: type="system", subtype="task_started".
type TaskStartedMessage struct {
	Type         string `json:"type"`    // "system"
	Subtype      string `json:"subtype"` // "task_started"
	TaskID       string `json:"task_id"`
	Description  string `json:"description,omitempty"`
	TaskType     string `json:"task_type,omitempty"`     // e.g., "agent"
	WorkflowName string `json:"workflow_name,omitempty"` // Workflow that spawned the task
	Prompt       string `json:"prompt,omitempty"`        // Initial prompt for the task
	ToolUseID    string `json:"tool_use_id,omitempty"`   // Parent tool use that spawned the task
	SessionID    string `json:"session_id"`
	UUID         string `json:"uuid"`
}

// TaskProgressMessage is emitted periodically while a background task runs.
// Wire format: type="system", subtype="task_progress".
type TaskProgressMessage struct {
	Type         string     `json:"type"`    // "system"
	Subtype      string     `json:"subtype"` // "task_progress"
	TaskID       string     `json:"task_id"`
	Description  string     `json:"description,omitempty"`
	LastToolName string     `json:"last_tool_name,omitempty"` // Most recent tool used
	Summary      string     `json:"summary,omitempty"`        // Progress summary text
	ToolUseID    string     `json:"tool_use_id,omitempty"`    // Parent tool use that spawned the task
	SessionID    string     `json:"session_id"`
	UUID         string     `json:"uuid"`
	Usage        *TaskUsage `json:"usage,omitempty"`
}

// TaskNotificationMessage is emitted when a background task completes, fails, or is stopped.
// Wire format: type="system", subtype="task_notification".
type TaskNotificationMessage struct {
	Type       string     `json:"type"`    // "system"
	Subtype    string     `json:"subtype"` // "task_notification"
	TaskID     string     `json:"task_id"`
	Status     string     `json:"status"` // "completed", "failed", "stopped"
	OutputFile string     `json:"output_file,omitempty"`
	Summary    string     `json:"summary,omitempty"`
	ToolUseID  string     `json:"tool_use_id,omitempty"` // Parent tool use that spawned the task
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

// =============================================================================
// System subtype message types
// =============================================================================

// InitMessage is the system init message emitted at session start.
// Contains session configuration: model, tools, permissions, etc.
type InitMessage struct {
	Type             string   `json:"type"`              // "system"
	Subtype          string   `json:"subtype"`           // "init"
	Agents           []string `json:"agents,omitempty"`
	APIKeySource     string   `json:"apiKeySource,omitempty"`
	Betas            []string `json:"betas,omitempty"`
	ClaudeCodeVersion string  `json:"claude_code_version,omitempty"`
	Cwd              string   `json:"cwd,omitempty"`
	Tools            []string `json:"tools,omitempty"`
	McpServers       []string `json:"mcp_servers,omitempty"`
	Model            string   `json:"model,omitempty"`
	PermissionMode   string   `json:"permissionMode,omitempty"`
	SlashCommands    []string `json:"slash_commands,omitempty"`
	OutputStyle      string   `json:"output_style,omitempty"`
	Skills           []string `json:"skills,omitempty"`
	Plugins          []string `json:"plugins,omitempty"`
	FastModeState    string   `json:"fast_mode_state,omitempty"` // "off", "on", "cooldown"
	UUID             string   `json:"uuid"`
	SessionID        string   `json:"session_id"`
}

// APIRetryMessage is emitted when an API request fails with a retryable error.
type APIRetryMessage struct {
	Type         string `json:"type"`    // "system"
	Subtype      string `json:"subtype"` // "api_retry"
	Attempt      int    `json:"attempt"`
	MaxRetries   int    `json:"max_retries"`
	RetryDelayMs int    `json:"retry_delay_ms"`
	ErrorStatus  *int   `json:"error_status"` // null for connection errors
	Error        string `json:"error"`
	UUID         string `json:"uuid"`
	SessionID    string `json:"session_id"`
}

// CompactBoundaryMessage marks a compaction event in the conversation.
type CompactBoundaryMessage struct {
	Type            string          `json:"type"`    // "system"
	Subtype         string          `json:"subtype"` // "compact_boundary"
	CompactMetadata CompactMetadata `json:"compact_metadata"`
	UUID            string          `json:"uuid"`
	SessionID       string          `json:"session_id"`
}

// CompactMetadata contains details about a compaction event.
type CompactMetadata struct {
	Trigger          string            `json:"trigger"` // "manual" or "auto"
	PreTokens        int               `json:"pre_tokens"`
	PreservedSegment *PreservedSegment `json:"preserved_segment,omitempty"`
}

// PreservedSegment describes the message range preserved during compaction.
type PreservedSegment struct {
	HeadUUID   string `json:"head_uuid"`
	AnchorUUID string `json:"anchor_uuid"`
	TailUUID   string `json:"tail_uuid"`
}

// HookStartedMessage is emitted when a hook begins execution.
type HookStartedMessage struct {
	Type      string `json:"type"`    // "system"
	Subtype   string `json:"subtype"` // "hook_started"
	HookID    string `json:"hook_id"`
	HookName  string `json:"hook_name"`
	HookEvent string `json:"hook_event"`
	UUID      string `json:"uuid"`
	SessionID string `json:"session_id"`
}

// HookProgressMessage is emitted during hook execution with output.
type HookProgressMessage struct {
	Type      string `json:"type"`    // "system"
	Subtype   string `json:"subtype"` // "hook_progress"
	HookID    string `json:"hook_id"`
	HookName  string `json:"hook_name"`
	HookEvent string `json:"hook_event"`
	Stdout    string `json:"stdout"`
	Stderr    string `json:"stderr"`
	Output    string `json:"output"`
	UUID      string `json:"uuid"`
	SessionID string `json:"session_id"`
}

// HookResponseMessage is emitted when a hook completes.
type HookResponseMessage struct {
	Type      string `json:"type"`    // "system"
	Subtype   string `json:"subtype"` // "hook_response"
	HookID    string `json:"hook_id"`
	HookName  string `json:"hook_name"`
	HookEvent string `json:"hook_event"`
	Output    string `json:"output"`
	Stdout    string `json:"stdout"`
	Stderr    string `json:"stderr"`
	ExitCode  *int   `json:"exit_code,omitempty"`
	Outcome   string `json:"outcome"` // "success", "error", "cancelled"
	UUID      string `json:"uuid"`
	SessionID string `json:"session_id"`
}

// LocalCommandOutputMessage contains output from a local slash command.
type LocalCommandOutputMessage struct {
	Type      string `json:"type"`    // "system"
	Subtype   string `json:"subtype"` // "local_command_output"
	Content   string `json:"content"`
	UUID      string `json:"uuid"`
	SessionID string `json:"session_id"`
}

// PersistedFile describes a file that was persisted during the session.
type PersistedFile struct {
	Filename string `json:"filename"`
	FileID   string `json:"file_id"`
}

// FailedFile describes a file that failed to persist.
type FailedFile struct {
	Filename string `json:"filename"`
	Error    string `json:"error"`
}

// FilesPersistedEvent is emitted when files are persisted to storage.
type FilesPersistedEvent struct {
	Type        string          `json:"type"`    // "system"
	Subtype     string          `json:"subtype"` // "files_persisted"
	Files       []PersistedFile `json:"files"`
	Failed      []FailedFile    `json:"failed"`
	ProcessedAt string          `json:"processed_at"`
	UUID        string          `json:"uuid"`
	SessionID   string          `json:"session_id"`
}

// ElicitationCompleteMessage is emitted when an MCP elicitation finishes.
type ElicitationCompleteMessage struct {
	Type          string `json:"type"`    // "system"
	Subtype       string `json:"subtype"` // "elicitation_complete"
	McpServerName string `json:"mcp_server_name"`
	ElicitationID string `json:"elicitation_id"`
	UUID          string `json:"uuid"`
	SessionID     string `json:"session_id"`
}

// SessionStateChangedMessage is emitted when the session state changes.
type SessionStateChangedMessage struct {
	Type      string `json:"type"`    // "system"
	Subtype   string `json:"subtype"` // "session_state_changed"
	State     string `json:"state"`
	UUID      string `json:"uuid"`
	SessionID string `json:"session_id"`
}

// StatusMessage is a system status update.
type StatusMessage struct {
	Type           string          `json:"type"`    // "system"
	Subtype        string          `json:"subtype"` // "status"
	Status         *string         `json:"status"`  // "compacting" or null
	PermissionMode *PermissionMode `json:"permissionMode,omitempty"`
	UUID           string          `json:"uuid"`
	SessionID      string          `json:"session_id"`
}

// =============================================================================
// Other top-level message types
// =============================================================================

// UserMessageReplay represents a replayed user message from a resumed session.
// Wire format: type="user" with isReplay=true.
type UserMessageReplay struct {
	Type            string         `json:"type"` // "user"
	Message         UserAPIMessage `json:"message"`
	ParentToolUseID *string        `json:"parent_tool_use_id"`
	IsReplay        bool           `json:"isReplay"`
	FileAttachments []any          `json:"file_attachments,omitempty"`
	IsSynthetic     bool           `json:"isSynthetic,omitempty"`
	ToolUseResult   any            `json:"tool_use_result,omitempty"`
	Priority        string         `json:"priority,omitempty"`
	Timestamp       string         `json:"timestamp,omitempty"`
	UUID            string         `json:"uuid"`
	SessionID       string         `json:"session_id"`
}

// AuthStatusMessage is emitted during authentication flows.
type AuthStatusMessage struct {
	Type             string   `json:"type"` // "auth_status"
	IsAuthenticating bool     `json:"isAuthenticating"`
	Output           []string `json:"output"`
	Error            string   `json:"error,omitempty"`
	UUID             string   `json:"uuid"`
	SessionID        string   `json:"session_id"`
}

// ToolProgressMessage is emitted during tool execution to report progress.
type ToolProgressMessage struct {
	Type               string  `json:"type"` // "tool_progress"
	ToolUseID          string  `json:"tool_use_id"`
	ToolName           string  `json:"tool_name"`
	ParentToolUseID    *string `json:"parent_tool_use_id"`
	ElapsedTimeSeconds float64 `json:"elapsed_time_seconds"`
	TaskID             string  `json:"task_id,omitempty"`
	UUID               string  `json:"uuid"`
	SessionID          string  `json:"session_id"`
}

// ToolUseSummaryMessage is a streamlined summary of tool call executions.
type ToolUseSummaryMessage struct {
	Type                 string   `json:"type"` // "tool_use_summary"
	Summary              string   `json:"summary"`
	PrecedingToolUseIDs  []string `json:"preceding_tool_use_ids"`
	UUID                 string   `json:"uuid"`
	SessionID            string   `json:"session_id"`
}

// PromptSuggestionMessage contains an AI-generated suggested next prompt.
type PromptSuggestionMessage struct {
	Type       string `json:"type"` // "prompt_suggestion"
	Suggestion string `json:"suggestion"`
	UUID       string `json:"uuid"`
	SessionID  string `json:"session_id"`
}

// =============================================================================
// Usage types
// =============================================================================

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
