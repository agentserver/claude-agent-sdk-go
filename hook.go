package agentsdk

import (
	"context"
	"time"
)

// HookEvent identifies when a hook fires in the agent lifecycle.
type HookEvent string

const (
	HookPreToolUse         HookEvent = "PreToolUse"
	HookPostToolUse        HookEvent = "PostToolUse"
	HookPostToolUseFailure HookEvent = "PostToolUseFailure"
	HookUserPromptSubmit   HookEvent = "UserPromptSubmit"
	HookSessionStart       HookEvent = "SessionStart"
	HookSessionEnd         HookEvent = "SessionEnd"
	HookStop               HookEvent = "Stop"
	HookStopFailure        HookEvent = "StopFailure"
	HookSubagentStart      HookEvent = "SubagentStart"
	HookSubagentStop       HookEvent = "SubagentStop"
	HookPreCompact         HookEvent = "PreCompact"
	HookPostCompact        HookEvent = "PostCompact"
	HookNotification       HookEvent = "Notification"
	HookPermissionRequest  HookEvent = "PermissionRequest"
	HookPermissionDenied   HookEvent = "PermissionDenied"
	HookSetup              HookEvent = "Setup"
	HookInstructionsLoaded HookEvent = "InstructionsLoaded"
	HookElicitation        HookEvent = "Elicitation"
	HookElicitationResult  HookEvent = "ElicitationResult"
	HookTeammateIdle       HookEvent = "TeammateIdle"
	HookTaskCreated        HookEvent = "TaskCreated"
	HookTaskCompleted      HookEvent = "TaskCompleted"
	HookConfigChange       HookEvent = "ConfigChange"
	HookCwdChanged         HookEvent = "CwdChanged"
	HookFileChanged        HookEvent = "FileChanged"
	HookWorktreeCreate     HookEvent = "WorktreeCreate"
	HookWorktreeRemove     HookEvent = "WorktreeRemove"
)

// HookCallback is a function invoked when a hook fires.
// It receives the hook input and returns a hook output.
type HookCallback func(ctx context.Context, input HookInput, toolUseID string) (HookOutput, error)

// HookMatcher defines which events trigger which callbacks.
type HookMatcher struct {
	Matcher string         // Regex pattern to match (e.g., tool name pattern "Bash|Write|Edit")
	Hooks   []HookCallback // Callbacks to invoke when matched
	Timeout time.Duration  // Timeout for hook execution (default 60s if zero)
}

// HookInput is the data passed to a hook callback.
// Fields are populated based on the hook event type.
type HookInput struct {
	// Common fields (present on all events).
	HookEventName  string `json:"hook_event_name"`
	SessionID      string `json:"session_id"`
	TranscriptPath string `json:"transcript_path"`
	Cwd            string `json:"cwd"`
	PermissionMode string `json:"permission_mode,omitempty"`
	AgentID        string `json:"agent_id,omitempty"`
	AgentType      string `json:"agent_type,omitempty"`

	// SessionStart fields.
	Source string `json:"source,omitempty"` // "startup", "resume", "clear", "compact"

	// SessionEnd fields.
	ExitReason string `json:"exit_reason,omitempty"` // "completed", "interrupted", "error"

	// Tool-specific fields (PreToolUse, PostToolUse, PostToolUseFailure).
	ToolName   string         `json:"tool_name,omitempty"`
	ToolInput  map[string]any `json:"tool_input,omitempty"`
	ToolResult any            `json:"tool_result,omitempty"` // PostToolUse only
	ToolUseID  string         `json:"tool_use_id,omitempty"`

	// UserPromptSubmit fields.
	Prompt string `json:"prompt,omitempty"`

	// Stop / SubagentStop fields.
	StopHookActive       bool   `json:"stop_hook_active,omitempty"`
	AgentTranscriptPath  string `json:"agent_transcript_path,omitempty"`  // SubagentStop only
	LastAssistantMessage string `json:"last_assistant_message,omitempty"` // SubagentStop only

	// PreCompact fields.
	Trigger            string `json:"trigger,omitempty"`             // "manual" or "auto"
	CustomInstructions string `json:"custom_instructions,omitempty"`

	// Notification fields.
	Message          string `json:"message,omitempty"`
	Title            string `json:"title,omitempty"`
	NotificationType string `json:"notification_type,omitempty"`

	// PermissionRequest fields.
	PermissionSuggestions []map[string]any `json:"permission_suggestions,omitempty"`

	// PostToolUseFailure fields.
	Error       string `json:"error,omitempty"`
	IsInterrupt bool   `json:"is_interrupt,omitempty"`

	// TeammateIdle fields.
	IdleDurationMs int64  `json:"idle_duration_ms,omitempty"`
	TeammateID     string `json:"teammate_id,omitempty"`

	// TaskCompleted / TaskCreated fields.
	TaskID     string `json:"task_id,omitempty"`
	TaskStatus string `json:"task_status,omitempty"` // "completed", "failed", "stopped"
	OutputFile string `json:"output_file,omitempty"`

	// ConfigChange fields.
	ConfigPath string `json:"config_path,omitempty"` // Path to changed config file
	ChangeType string `json:"change_type,omitempty"` // "created", "modified", "deleted"

	// WorktreeCreate / WorktreeRemove fields.
	WorktreePath   string `json:"worktree_path,omitempty"`
	WorktreeBranch string `json:"worktree_branch,omitempty"`

	// StopFailure fields.
	FailureReason string `json:"failure_reason,omitempty"`

	// PostCompact fields.
	TokensSaved int `json:"tokens_saved,omitempty"`

	// InstructionsLoaded fields.
	InstructionsPath string `json:"instructions_path,omitempty"`

	// Elicitation fields.
	ElicitationID string         `json:"elicitation_id,omitempty"`
	ServerName    string         `json:"server_name,omitempty"`
	Schema        map[string]any `json:"schema,omitempty"`

	// ElicitationResult fields.
	ElicitationResponse map[string]any `json:"elicitation_response,omitempty"`

	// CwdChanged fields.
	OldCwd string `json:"old_cwd,omitempty"`
	NewCwd string `json:"new_cwd,omitempty"`

	// FileChanged fields.
	FilePath       string `json:"file_path,omitempty"`
	FileChangeType string `json:"file_change_type,omitempty"`

	// PermissionDenied fields.
	DeniedToolName string `json:"denied_tool_name,omitempty"`
	DeniedReason   string `json:"denied_reason,omitempty"`
}

// HookOutput is the response from a hook callback.
type HookOutput struct {
	Continue       bool   `json:"continue,omitempty"`       // Continue execution after hook
	SuppressOutput bool   `json:"suppressOutput,omitempty"` // Suppress output from this hook
	SystemMessage  string `json:"systemMessage,omitempty"`  // Inject a system message into context
	Reason         string `json:"reason,omitempty"`         // Debug reason for Claude
	Decision       string `json:"decision,omitempty"`       // "block" for certain hooks
	StopReason     string `json:"stopReason,omitempty"`     // Reason for stopping execution

	// Async output — return immediately without blocking the agent loop.
	// Use for fire-and-forget side effects (logging, webhooks).
	Async        bool `json:"async,omitempty"`        // Fire-and-forget mode
	AsyncTimeout int  `json:"asyncTimeout,omitempty"` // Timeout in milliseconds

	// HookSpecificOutput contains event-specific response data.
	HookSpecificOutput *HookSpecificOutput `json:"hookSpecificOutput,omitempty"`
}

// HookSpecificOutput contains event-specific fields in a hook response.
type HookSpecificOutput struct {
	HookEventName            string         `json:"hookEventName"`
	PermissionDecision       string         `json:"permissionDecision,omitempty"`       // "allow", "deny", "ask"
	PermissionDecisionReason string         `json:"permissionDecisionReason,omitempty"` // Human-readable reason
	UpdatedInput             map[string]any `json:"updatedInput,omitempty"`             // Modify tool input (PreToolUse allow)
	AdditionalContext        string         `json:"additionalContext,omitempty"`        // Extra context for Claude
	UpdatedMCPToolOutput     map[string]any `json:"updatedMCPToolOutput,omitempty"`     // Modify MCP tool output (PostToolUse)
}
