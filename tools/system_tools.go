package tools

// =============================================================================
// Bash
// =============================================================================

// BashInput is the input for the Bash tool.
type BashInput struct {
	Command                string `json:"command"`
	Timeout                *int   `json:"timeout,omitempty"` // milliseconds, max 600000
	Description            string `json:"description,omitempty"`
	RunInBackground        bool   `json:"run_in_background,omitempty"`
	DangerouslyDisableSandbox bool `json:"dangerouslyDisableSandbox,omitempty"`
}

// BashOutput is the result of a Bash tool execution.
type BashOutput struct {
	Stdout                    string `json:"stdout"`
	Stderr                    string `json:"stderr"`
	Interrupted               bool   `json:"interrupted"`
	RawOutputPath             string `json:"rawOutputPath,omitempty"`
	IsImage                   bool   `json:"isImage,omitempty"`
	BackgroundTaskID          string `json:"backgroundTaskId,omitempty"`
	BackgroundedByUser        bool   `json:"backgroundedByUser,omitempty"`
	AssistantAutoBackgrounded bool   `json:"assistantAutoBackgrounded,omitempty"`
	DangerouslyDisableSandbox bool   `json:"dangerouslyDisableSandbox,omitempty"`
	ReturnCodeInterpretation  string `json:"returnCodeInterpretation,omitempty"`
	NoOutputExpected          bool   `json:"noOutputExpected,omitempty"`
	StructuredContent         []any  `json:"structuredContent,omitempty"`
	PersistedOutputPath       string `json:"persistedOutputPath,omitempty"`
	PersistedOutputSize       *int   `json:"persistedOutputSize,omitempty"`
	StaleReadFileStateHint    string `json:"staleReadFileStateHint,omitempty"`
}

// =============================================================================
// Agent
// =============================================================================

// AgentInput is the input for the Agent tool.
type AgentInput struct {
	Description     string `json:"description"`
	Prompt          string `json:"prompt"`
	SubagentType    string `json:"subagent_type,omitempty"`
	Model           string `json:"model,omitempty"`            // "sonnet", "opus", "haiku"
	Resume          string `json:"resume,omitempty"`           // Resume an existing agent session by ID
	RunInBackground bool   `json:"run_in_background,omitempty"`
	MaxTurns        *int   `json:"max_turns,omitempty"`        // Maximum agentic turns before stopping
	Name            string `json:"name,omitempty"`
	TeamName        string `json:"team_name,omitempty"`
	Mode            string `json:"mode,omitempty"`             // "acceptEdits", "bypassPermissions", "default", "dontAsk", "plan"
	Isolation       string `json:"isolation,omitempty"`        // "worktree"
}

// AgentOutputCompleted is returned when an agent completes synchronously.
type AgentOutputCompleted struct {
	AgentID           string        `json:"agentId"`
	AgentType         string        `json:"agentType,omitempty"`
	Content           []TextContent `json:"content"`
	TotalToolUseCount int           `json:"totalToolUseCount"`
	TotalDurationMs   int           `json:"totalDurationMs"`
	TotalTokens       int           `json:"totalTokens"`
	Usage             Usage         `json:"usage"`
	Status            string        `json:"status"` // "completed"
	Prompt            string        `json:"prompt"`
}

// AgentOutputAsync is returned when an agent is launched in the background.
type AgentOutputAsync struct {
	Status            string `json:"status"` // "async_launched"
	AgentID           string `json:"agentId"`
	Description       string `json:"description"`
	Prompt            string `json:"prompt"`
	OutputFile        string `json:"outputFile"`
	CanReadOutputFile bool   `json:"canReadOutputFile,omitempty"`
}

// AgentOutputSubAgentEntered is returned when an interactive sub-agent is entered.
type AgentOutputSubAgentEntered struct {
	Status      string `json:"status"` // "sub_agent_entered"
	Description string `json:"description"`
	Message     string `json:"message"`
}

// =============================================================================
// WebFetch
// =============================================================================

// WebFetchInput is the input for the WebFetch tool.
type WebFetchInput struct {
	URL    string `json:"url"`
	Prompt string `json:"prompt"`
}

// WebFetchOutput is the result of a WebFetch tool execution.
type WebFetchOutput struct {
	Bytes      int    `json:"bytes"`
	Code       int    `json:"code"`
	CodeText   string `json:"codeText"`
	Result     string `json:"result"`
	DurationMs int    `json:"durationMs"`
	URL        string `json:"url"`
}

// =============================================================================
// WebSearch
// =============================================================================

// WebSearchInput is the input for the WebSearch tool.
type WebSearchInput struct {
	Query          string   `json:"query"`
	AllowedDomains []string `json:"allowed_domains,omitempty"`
	BlockedDomains []string `json:"blocked_domains,omitempty"`
}

// WebSearchOutput is the result of a WebSearch tool execution.
type WebSearchOutput struct {
	Query           string `json:"query"`
	Results         []any  `json:"results"` // Mixed: search result objects or string commentary
	DurationSeconds float64 `json:"durationSeconds"`
}

// =============================================================================
// AskUserQuestion
// =============================================================================

// AskUserQuestionInput is the input for the AskUserQuestion tool.
type AskUserQuestionInput struct {
	Questions []QuestionInput `json:"questions"`
}

// QuestionInput defines a single question in AskUserQuestion.
type QuestionInput struct {
	Question    string        `json:"question"`
	Header      string        `json:"header"` // max 12 chars
	Options     []OptionInput `json:"options"` // 2-4 options
	MultiSelect bool          `json:"multiSelect"`
}

// OptionInput defines a single option in a question.
type OptionInput struct {
	Label       string `json:"label"`
	Description string `json:"description"`
	Preview     string `json:"preview,omitempty"`
}

// AskUserQuestionOutput is the result of an AskUserQuestion execution.
type AskUserQuestionOutput struct {
	Questions   []QuestionInput        `json:"questions"`
	Answers     map[string]string      `json:"answers"`
	Annotations map[string]Annotation  `json:"annotations,omitempty"`
}

// Annotation contains per-question metadata from the user.
type Annotation struct {
	Preview string `json:"preview,omitempty"`
	Notes   string `json:"notes,omitempty"`
}

// =============================================================================
// Config
// =============================================================================

// ConfigInput is the input for the Config tool.
type ConfigInput struct {
	Setting string `json:"setting"`
	Value   any    `json:"value,omitempty"` // string, bool, or number
}

// ConfigOutput is the result of a Config tool execution.
type ConfigOutput struct {
	Success       bool   `json:"success"`
	Operation     string `json:"operation,omitempty"` // "get" or "set"
	Setting       string `json:"setting,omitempty"`
	Value         any    `json:"value,omitempty"`
	PreviousValue any    `json:"previousValue,omitempty"`
	NewValue      any    `json:"newValue,omitempty"`
	Error         string `json:"error,omitempty"`
}

// =============================================================================
// Worktree
// =============================================================================

// EnterWorktreeInput is the input for the EnterWorktree tool.
type EnterWorktreeInput struct {
	Name string `json:"name,omitempty"`
}

// EnterWorktreeOutput is the result of EnterWorktree execution.
type EnterWorktreeOutput struct {
	WorktreePath   string `json:"worktreePath"`
	WorktreeBranch string `json:"worktreeBranch,omitempty"`
	Message        string `json:"message"`
}

// ExitWorktreeInput is the input for the ExitWorktree tool.
type ExitWorktreeInput struct {
	Action         string `json:"action"`                    // "keep" or "remove"
	DiscardChanges bool   `json:"discard_changes,omitempty"`
}

// ExitWorktreeOutput is the result of ExitWorktree execution.
type ExitWorktreeOutput struct {
	Action           string `json:"action"` // "keep" or "remove"
	OriginalCwd      string `json:"originalCwd"`
	WorktreePath     string `json:"worktreePath"`
	WorktreeBranch   string `json:"worktreeBranch,omitempty"`
	TmuxSessionName  string `json:"tmuxSessionName,omitempty"`
	DiscardedFiles   *int   `json:"discardedFiles,omitempty"`
	DiscardedCommits *int   `json:"discardedCommits,omitempty"`
	Message          string `json:"message"`
}

// =============================================================================
// ExitPlanMode
// =============================================================================

// ExitPlanModeInput is the input for the ExitPlanMode tool.
type ExitPlanModeInput struct {
	AllowedPrompts []AllowedPrompt `json:"allowedPrompts,omitempty"`
}

// AllowedPrompt describes a prompt-based permission needed for plan execution.
type AllowedPrompt struct {
	Tool   string `json:"tool"` // "Bash"
	Prompt string `json:"prompt"`
}

// ExitPlanModeOutput is the result of ExitPlanMode execution.
type ExitPlanModeOutput struct {
	Plan                   *string `json:"plan"`
	IsAgent                bool    `json:"isAgent"`
	FilePath               string  `json:"filePath,omitempty"`
	HasTaskTool            bool    `json:"hasTaskTool,omitempty"`
	PlanWasEdited          bool    `json:"planWasEdited,omitempty"`
	AwaitingLeaderApproval bool    `json:"awaitingLeaderApproval,omitempty"`
	RequestID              string  `json:"requestId,omitempty"`
}

// =============================================================================
// Task management
// =============================================================================

// TaskOutputInput is the input for the TaskOutput tool.
type TaskOutputInput struct {
	TaskID  string `json:"task_id"`
	Block   bool   `json:"block"`
	Timeout int    `json:"timeout"` // milliseconds
}

// TaskStopInput is the input for the TaskStop tool.
type TaskStopInput struct {
	TaskID  string `json:"task_id,omitempty"`
	ShellID string `json:"shell_id,omitempty"` // deprecated
}

// TaskStopOutput is the result of TaskStop execution.
type TaskStopOutput struct {
	Message  string `json:"message"`
	TaskID   string `json:"task_id"`
	TaskType string `json:"task_type"`
	Command  string `json:"command,omitempty"`
}

// =============================================================================
// TodoWrite
// =============================================================================

// TodoWriteInput is the input for the TodoWrite tool.
type TodoWriteInput struct {
	Todos []TodoItem `json:"todos"`
}

// TodoItem represents a single todo item.
type TodoItem struct {
	Content    string `json:"content"`
	Status     string `json:"status"`     // "pending", "in_progress", "completed"
	ActiveForm string `json:"activeForm"`
}

// TodoWriteOutput is the result of a TodoWrite execution.
type TodoWriteOutput struct {
	OldTodos                []TodoItem `json:"oldTodos"`
	NewTodos                []TodoItem `json:"newTodos"`
	VerificationNudgeNeeded bool       `json:"verificationNudgeNeeded,omitempty"`
}

// =============================================================================
// MCP resources
// =============================================================================

// ListMcpResourcesInput is the input for the ListMcpResources tool.
type ListMcpResourcesInput struct {
	Server string `json:"server,omitempty"`
}

// McpResource describes a resource exposed by an MCP server.
type McpResource struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	MimeType    string `json:"mimeType,omitempty"`
	Description string `json:"description,omitempty"`
	Server      string `json:"server"`
}

// ListMcpResourcesOutput is the result of ListMcpResources execution.
type ListMcpResourcesOutput = []McpResource

// ReadMcpResourceInput is the input for the ReadMcpResource tool.
type ReadMcpResourceInput struct {
	Server string `json:"server"`
	URI    string `json:"uri"`
}

// ReadMcpResourceOutput is the result of ReadMcpResource execution.
type ReadMcpResourceOutput struct {
	Contents []McpResourceContent `json:"contents"`
}

// McpResourceContent is a single content item from an MCP resource.
type McpResourceContent struct {
	URI          string `json:"uri"`
	MimeType     string `json:"mimeType,omitempty"`
	Text         string `json:"text,omitempty"`
	BlobSavedTo  string `json:"blobSavedTo,omitempty"`
}
