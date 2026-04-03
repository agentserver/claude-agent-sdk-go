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
	ExitReason string `json:"reason,omitempty"` // SessionEnd: "clear", "resume", "logout", etc. Also receives PermissionDenied.reason (free-form string).

	// Tool-specific fields (PreToolUse, PostToolUse, PostToolUseFailure).
	ToolName   string         `json:"tool_name,omitempty"`
	ToolInput  map[string]any `json:"tool_input,omitempty"`
	ToolResponse any            `json:"tool_response,omitempty"` // PostToolUse only
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

	// PostCompact fields.
	CompactSummary string `json:"compact_summary,omitempty"` // Summary produced by compaction

	// Notification fields.
	Message          string `json:"message,omitempty"`
	Title            string `json:"title,omitempty"`
	NotificationType string `json:"notification_type,omitempty"`

	// PermissionRequest fields.
	PermissionSuggestions []map[string]any `json:"permission_suggestions,omitempty"`

	// PostToolUseFailure / StopFailure fields.
	// Both events send {"error": string} with the same JSON key.
	Error       string `json:"error,omitempty"`
	IsInterrupt bool   `json:"is_interrupt,omitempty"` // PostToolUseFailure only

	// TeammateIdle fields.
	TeammateName string `json:"teammate_name,omitempty"`
	TeamName     string `json:"team_name,omitempty"`

	// TaskCreated / TaskCompleted fields.
	TaskID          string `json:"task_id,omitempty"`
	TaskSubject     string `json:"task_subject,omitempty"`
	TaskDescription string `json:"task_description,omitempty"`

	// ConfigChange fields.
	// For ConfigChange events, Source is the settings layer that changed
	// ("user_settings", "project_settings", "local_settings", "policy_settings", "skills").
	// Note: shares the json:"source" tag with SessionStart's Source field above.
	FilePath string `json:"file_path,omitempty"` // Path to changed file (ConfigChange, InstructionsLoaded, FileChanged)

	// WorktreeCreate / WorktreeRemove fields.
	WorktreePath   string `json:"worktree_path,omitempty"`
	WorktreeBranch string `json:"worktree_branch,omitempty"`

	// StopFailure fields (in addition to shared Error above).
	ErrorDetails string `json:"error_details,omitempty"` // Additional error context

	// InstructionsLoaded fields.
	MemoryType      string   `json:"memory_type,omitempty"`       // "User", "Project", "Local", "Managed"
	LoadReason      string   `json:"load_reason,omitempty"`       // "session_start", "nested_traversal", "path_glob_match", "include", "compact"
	Globs           []string `json:"globs,omitempty"`             // Glob patterns that matched
	TriggerFilePath string   `json:"trigger_file_path,omitempty"` // File that triggered loading
	ParentFilePath  string   `json:"parent_file_path,omitempty"`  // Parent file for nested loads

	// Elicitation fields.
	ElicitationID   string         `json:"elicitation_id,omitempty"`
	McpServerName   string         `json:"mcp_server_name,omitempty"`
	ElicitationMode string         `json:"mode,omitempty"`            // "form" or "url"
	URL             string         `json:"url,omitempty"`
	RequestedSchema map[string]any `json:"requested_schema,omitempty"`

	// ElicitationResult fields.
	Action            string         `json:"action,omitempty"`  // "accept", "decline", "cancel"
	ElicitationContent map[string]any `json:"content,omitempty"` // Response content

	// CwdChanged fields.
	OldCwd string `json:"old_cwd,omitempty"`
	NewCwd string `json:"new_cwd,omitempty"`

	// FileChanged fields.
	FileEvent string `json:"event,omitempty"` // "change", "add", "unlink"
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
	PermissionDecision       string         `json:"permissionDecision,omitempty"`       // "allow", "deny", "ask" (PreToolUse)
	PermissionDecisionReason string         `json:"permissionDecisionReason,omitempty"` // Human-readable reason (PreToolUse)
	UpdatedInput             map[string]any `json:"updatedInput,omitempty"`             // Modify tool input (PreToolUse allow)
	AdditionalContext        string         `json:"additionalContext,omitempty"`        // Extra context for Claude
	UpdatedMCPToolOutput     map[string]any `json:"updatedMCPToolOutput,omitempty"`     // Modify MCP tool output (PostToolUse)
	InitialUserMessage       string         `json:"initialUserMessage,omitempty"`       // Override initial prompt (SessionStart)
	WatchPaths               []string       `json:"watchPaths,omitempty"`               // File paths to watch (SessionStart, CwdChanged, FileChanged)
	WorktreePath             string         `json:"worktreePath,omitempty"`             // Created worktree path (WorktreeCreate)
	Retry                    bool           `json:"retry,omitempty"`                    // Retry denied tool (PermissionDenied)

	// PermissionRequest hook: decision object with allow/deny discriminant.
	// Use PermissionRequestDecisionAllow or PermissionRequestDecisionDeny helpers.
	Decision *PermissionRequestDecision `json:"decision,omitempty"`
}

// PermissionRequestDecision is the decision structure for PermissionRequest hooks.
// Set either the Allow or Deny variant fields based on the Behavior value.
type PermissionRequestDecision struct {
	Behavior           string             `json:"behavior"`                     // "allow" or "deny"
	UpdatedInput       map[string]any     `json:"updatedInput,omitempty"`       // Allow: modified tool input
	UpdatedPermissions []PermissionUpdate `json:"updatedPermissions,omitempty"` // Allow: dynamic permission rule changes
	Message            string             `json:"message,omitempty"`            // Deny: reason message
	Interrupt          bool               `json:"interrupt,omitempty"`          // Deny: stop execution
}
