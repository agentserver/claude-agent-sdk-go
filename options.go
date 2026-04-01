package agentsdk

import (
	"context"
	"time"
)

// PermissionMode controls how tool permissions are handled.
type PermissionMode string

const (
	PermissionDefault     PermissionMode = "default"
	PermissionAcceptEdits PermissionMode = "acceptEdits"
	PermissionPlan        PermissionMode = "plan"
	PermissionBypassAll   PermissionMode = "bypassPermissions"
	PermissionDontAsk     PermissionMode = "dontAsk"
)

// PreviewFormat controls how AskUserQuestion previews are rendered.
type PreviewFormat string

const (
	PreviewFormatMarkdown PreviewFormat = "markdown"
	PreviewFormatHTML     PreviewFormat = "html"
)

// ToolConfig configures tool-specific behavior.
type ToolConfig struct {
	AskUserQuestion *AskUserQuestionConfig `json:"askUserQuestion,omitempty"`
}

// AskUserQuestionConfig configures the AskUserQuestion tool.
type AskUserQuestionConfig struct {
	PreviewFormat PreviewFormat `json:"previewFormat,omitempty"` // "markdown" or "html"
}

// Effort controls how hard Claude tries (maps to --effort flag).
type Effort string

const (
	EffortLow    Effort = "low"
	EffortMedium Effort = "medium"
	EffortHigh   Effort = "high"
	EffortMax    Effort = "max"
)

// SdkBeta identifies an opt-in beta feature.
type SdkBeta = string

const (
	// SdkBetaContext1M enables the 1M context window beta.
	SdkBetaContext1M SdkBeta = "context-1m-2025-08-07"
)

// ThinkingConfig configures extended thinking.
type ThinkingConfig struct {
	Type         string `json:"type"`                    // "enabled", "disabled", or "adaptive" (default for compatible models)
	BudgetTokens int    `json:"budget_tokens,omitempty"` // Max tokens for thinking (only for "enabled")
}

// SettingSource is a path to a Claude Code settings file.
type SettingSource string

// PluginConfig defines a plugin to load.
type PluginConfig struct {
	Name string `json:"name"`
	Path string `json:"path,omitempty"`
}

// TaskBudget configures token budgets for background tasks.
type TaskBudget struct {
	Total int `json:"total"`
}

// SandboxSettings configures filesystem and network isolation.
type SandboxSettings struct {
	Enabled                       bool                        `json:"enabled,omitempty"`
	FailIfUnavailable             bool                        `json:"failIfUnavailable,omitempty"`
	AutoAllowBashIfSandboxed      bool                        `json:"autoAllowBashIfSandboxed,omitempty"`
	AllowUnsandboxedCommands      bool                        `json:"allowUnsandboxedCommands,omitempty"`
	ExcludedCommands              []string                    `json:"excludedCommands,omitempty"`
	Network                       *SandboxNetworkConfig       `json:"network,omitempty"`
	Filesystem                    *SandboxFilesystemConfig    `json:"filesystem,omitempty"`
	IgnoreViolations              map[string][]string         `json:"ignoreViolations,omitempty"` // pattern category → patterns
	EnableWeakerNestedSandbox     bool                        `json:"enableWeakerNestedSandbox,omitempty"`
	EnableWeakerNetworkIsolation  bool                        `json:"enableWeakerNetworkIsolation,omitempty"`
	Ripgrep                       *SandboxRipgrepConfig       `json:"ripgrep,omitempty"`
}

// SandboxNetworkConfig configures network isolation for sandboxed sessions.
type SandboxNetworkConfig struct {
	AllowedDomains          []string `json:"allowedDomains,omitempty"`
	AllowManagedDomainsOnly bool     `json:"allowManagedDomainsOnly,omitempty"`
	AllowUnixSockets        []string `json:"allowUnixSockets,omitempty"`
	AllowAllUnixSockets     bool     `json:"allowAllUnixSockets,omitempty"`
	AllowLocalBinding       bool     `json:"allowLocalBinding,omitempty"`
	HTTPProxyPort           int      `json:"httpProxyPort,omitempty"`
	SOCKSProxyPort          int      `json:"socksProxyPort,omitempty"`
}

// SandboxFilesystemConfig configures filesystem access restrictions.
type SandboxFilesystemConfig struct {
	AllowWrite              []string `json:"allowWrite,omitempty"`
	DenyWrite               []string `json:"denyWrite,omitempty"`
	AllowRead               []string `json:"allowRead,omitempty"`
	DenyRead                []string `json:"denyRead,omitempty"`
	AllowManagedReadPathsOnly bool   `json:"allowManagedReadPathsOnly,omitempty"`
}

// SandboxRipgrepConfig configures a custom ripgrep binary for sandbox use.
type SandboxRipgrepConfig struct {
	Command string   `json:"command"`
	Args    []string `json:"args,omitempty"`
}

// AgentDefinition defines a subagent configuration.
type AgentDefinition struct {
	Description                        string         `json:"description"`
	Prompt                             string         `json:"prompt"`
	Tools                              []string       `json:"tools,omitempty"`
	DisallowedTools                    []string       `json:"disallowedTools,omitempty"`
	Model                              string         `json:"model,omitempty"`                              // "sonnet", "opus", "haiku", "inherit"
	McpServers                         []string       `json:"mcpServers,omitempty"`                         // Reference parent MCP server names
	Skills                             []string       `json:"skills,omitempty"`                             // Preload specialized skills
	MaxTurns                           int            `json:"maxTurns,omitempty"`
	InitialPrompt                      string         `json:"initialPrompt,omitempty"`                      // Prompt sent when agent starts
	Background                         bool           `json:"background,omitempty"`                         // Run as fire-and-forget background task
	Memory                             string         `json:"memory,omitempty"`                             // "user", "project", or "local"
	Effort                             Effort         `json:"effort,omitempty"`                             // Per-agent reasoning depth
	PermissionMode                     PermissionMode `json:"permissionMode,omitempty"`                     // Per-agent tool permission strategy
	CriticalSystemReminderExperimental string         `json:"criticalSystemReminder_EXPERIMENTAL,omitempty"` // Experimental: high-priority system reminder
}

// PermissionDecisionClassification classifies a permission decision for telemetry.
type PermissionDecisionClassification string

const (
	PermissionDecisionUserTemporary PermissionDecisionClassification = "user_temporary"
	PermissionDecisionUserPermanent PermissionDecisionClassification = "user_permanent"
	PermissionDecisionUserReject    PermissionDecisionClassification = "user_reject"
)

// PermissionResult is the response from a CanUseToolFunc callback.
type PermissionResult struct {
	Behavior                string                            `json:"behavior"`                               // "allow", "deny", or "ask" (delegate to CLI default prompt)
	Message                 string                            `json:"message,omitempty"`                      // Reason message (deny/ask)
	Interrupt               bool                              `json:"interrupt,omitempty"`                    // Stop execution (deny only)
	UpdatedInput            map[string]any                    `json:"updatedInput,omitempty"`                 // Modified tool input (allow only)
	UpdatedPermissions      []PermissionUpdate                `json:"updatedPermissions,omitempty"`           // Dynamic permission rule changes (allow only)
	ToolUseID               string                            `json:"toolUseID,omitempty"`                    // Tool use ID this decision applies to
	DecisionClassification  PermissionDecisionClassification  `json:"decisionClassification,omitempty"`       // Telemetry classification
}

// PermissionUpdate describes a dynamic change to permission rules.
type PermissionUpdate struct {
	Type        string           `json:"type"`                  // "addRules", "replaceRules", "removeRules", "setMode", "addDirectories", "removeDirectories"
	Rules       []PermissionRule `json:"rules,omitempty"`
	Behavior    string           `json:"behavior,omitempty"`    // "allow", "deny", "ask"
	Mode        PermissionMode   `json:"mode,omitempty"`        // For "setMode" type
	Directories []string         `json:"directories,omitempty"` // For "addDirectories"/"removeDirectories"
	Destination string           `json:"destination,omitempty"` // "userSettings", "projectSettings", "localSettings", "session"
}

// PermissionRule defines a single permission rule pattern.
type PermissionRule struct {
	ToolName    string `json:"toolName"`              // Tool name or pattern
	RuleContent string `json:"ruleContent,omitempty"` // Optional additional matching pattern
}

// ToolPermissionContext provides context for permission decisions.
type ToolPermissionContext struct {
	ToolUseID      string             `json:"tool_use_id"`
	AgentID        string             `json:"agent_id,omitempty"`
	Suggestions    []PermissionUpdate `json:"suggestions,omitempty"`     // CLI-provided permission update suggestions
	BlockedPath    string             `json:"blocked_path,omitempty"`    // File path that triggered the permission check
	DecisionReason string             `json:"decision_reason,omitempty"` // Reason for the permission check
	Title          string             `json:"title,omitempty"`           // Tool display title
	DisplayName    string             `json:"display_name,omitempty"`    // Tool display name
	Description    string             `json:"description,omitempty"`     // Tool description
}

// CanUseToolFunc is called when claude requests permission to use a tool.
// Return a PermissionResult to allow or deny the tool use.
// The context.Context is cancelled if the operation is aborted.
type CanUseToolFunc func(ctx context.Context, toolName string, toolInput map[string]any, options ToolPermissionContext) (*PermissionResult, error)

// ElicitationRequest describes an MCP elicitation request from the CLI.
type ElicitationRequest struct {
	ServerName      string         `json:"serverName"`
	Message         string         `json:"message"`
	Mode            string         `json:"mode,omitempty"`            // "form" or "url"
	URL             string         `json:"url,omitempty"`             // URL to open (url mode only)
	ElicitationID   string         `json:"elicitationId,omitempty"`   // Correlation ID (url mode only)
	RequestedSchema map[string]any `json:"requestedSchema,omitempty"` // JSON Schema (form mode only)
}

// ElicitationResult is the response to an MCP elicitation request.
type ElicitationResult struct {
	Action  string         `json:"action"`            // "accept", "decline", "cancel"
	Content map[string]any `json:"content,omitempty"` // Form values (accept only)
}

// OnElicitationFunc handles MCP elicitation requests from the CLI.
// Called when an MCP server requests user input (form fields, URL auth, etc.).
// If nil, elicitation requests are automatically declined.
type OnElicitationFunc func(ctx context.Context, req ElicitationRequest) (*ElicitationResult, error)

// queryConfig holds the resolved configuration for a Query or Client.
type queryConfig struct {
	// Model & Reasoning
	model             string
	fallbackModel     string
	thinking          *ThinkingConfig
	effort            Effort
	maxThinkingTokens *int // deprecated: use thinking

	// Tools & Permissions
	allowedTools                    []string
	disallowedTools                 []string
	tools                           []string
	toolsPreset                     bool
	permissionMode                  PermissionMode
	bypassPermissions               bool
	allowDangerouslySkipPermissions bool
	canUseTool                      CanUseToolFunc
	permissionPromptToolName        string

	// System Prompt
	systemPrompt       string
	systemPromptPreset string // append string for preset mode
	systemPromptFile   string

	// Conversation
	resumeSessionID string
	resumeSessionAt string // Resume from specific message UUID
	continueSession bool
	sessionID       string
	forkSession     bool
	maxTurns        int
	maxBudgetUSD    float64

	// Session persistence
	persistSession *bool // nil = default (true), false = in-memory only

	// Environment
	cwd            string
	env            map[string]string
	additionalDirs []string
	cliPath        string
	settingSources []SettingSource
	apiKey         string

	// Tool config
	toolConfig *ToolConfig

	// Extensions
	mcpServers map[string]McpServerConfig
	agents     map[string]AgentDefinition
	hooks      map[HookEvent][]HookMatcher
	plugins    []PluginConfig

	// Agent configuration
	agent string // Main thread agent name (--agent flag)

	// Budget
	taskBudget *TaskBudget

	// Sandbox
	sandbox *SandboxSettings

	// Identity
	user string

	// Extra CLI arguments (escape hatch)
	extraArgs []string

	// Output
	outputFormat           map[string]any // JSON schema for structured output
	includePartialMessages bool
	includeHookEvents      bool

	// Elicitation
	onElicitation OnElicitationFunc

	// Prompt suggestions
	promptSuggestions      *bool
	agentProgressSummaries *bool

	// Settings (inline object or file path)
	settings any // map[string]any or string

	// MCP config strictness
	strictMcpConfig bool

	// File checkpointing
	fileCheckpointing bool

	// Debugging
	stderrFunc    func(string)
	debug         bool
	debugFile     string
	maxBufferSize int

	// Beta features
	betas []SdkBeta

	// Timeouts
	processTimeout time.Duration
}

// QueryOption configures a Query or Client.
type QueryOption func(*queryConfig)

// --- Model & Reasoning ---

// WithModel sets the Claude model to use (e.g., "sonnet", "opus", "haiku").
func WithModel(model string) QueryOption {
	return func(c *queryConfig) { c.model = model }
}

// WithFallbackModel sets a fallback model if the primary is unavailable.
func WithFallbackModel(model string) QueryOption {
	return func(c *queryConfig) { c.fallbackModel = model }
}

// WithThinking enables extended thinking with the given configuration.
func WithThinking(config ThinkingConfig) QueryOption {
	return func(c *queryConfig) { c.thinking = &config }
}

// WithEffort sets the effort level for reasoning.
func WithEffort(effort Effort) QueryOption {
	return func(c *queryConfig) { c.effort = effort }
}

// WithMaxThinkingTokens sets the maximum thinking tokens for session initialization.
// Deprecated: Use WithThinking instead. On Opus 4.6, this is treated as on/off
// (0 = disabled, any other value = adaptive).
func WithMaxThinkingTokens(tokens int) QueryOption {
	return func(c *queryConfig) { c.maxThinkingTokens = &tokens }
}

// --- Tools & Permissions ---

// WithAllowedTools sets the list of tools the model may use.
func WithAllowedTools(tools ...string) QueryOption {
	return func(c *queryConfig) { c.allowedTools = tools }
}

// WithDisallowedTools sets tools the model may NOT use.
func WithDisallowedTools(tools ...string) QueryOption {
	return func(c *queryConfig) { c.disallowedTools = tools }
}

// WithTools sets explicit tool names to enable.
func WithTools(tools ...string) QueryOption {
	return func(c *queryConfig) { c.tools = tools }
}

// WithToolsPreset uses the Claude Code default tool set.
func WithToolsPreset() QueryOption {
	return func(c *queryConfig) { c.toolsPreset = true }
}

// WithPermissionMode sets the tool permission mode.
func WithPermissionMode(mode PermissionMode) QueryOption {
	return func(c *queryConfig) { c.permissionMode = mode }
}

// WithBypassPermissions bypasses all permission checks.
// Requires WithAllowDangerouslySkipPermissions to be set.
func WithBypassPermissions() QueryOption {
	return func(c *queryConfig) { c.bypassPermissions = true }
}

// WithAllowDangerouslySkipPermissions acknowledges the risk of bypassing permissions.
func WithAllowDangerouslySkipPermissions() QueryOption {
	return func(c *queryConfig) { c.allowDangerouslySkipPermissions = true }
}

// WithCanUseTool sets a callback for runtime permission decisions.
func WithCanUseTool(fn CanUseToolFunc) QueryOption {
	return func(c *queryConfig) { c.canUseTool = fn }
}

// WithPermissionPromptToolName sets a custom MCP tool name for permission prompts.
// The named tool will be called instead of the default CLI permission prompt.
func WithPermissionPromptToolName(name string) QueryOption {
	return func(c *queryConfig) { c.permissionPromptToolName = name }
}

// --- System Prompt ---

// WithSystemPrompt sets a custom system prompt (replaces default).
func WithSystemPrompt(prompt string) QueryOption {
	return func(c *queryConfig) { c.systemPrompt = prompt }
}

// WithSystemPromptPreset uses the Claude Code default prompt with appended text.
func WithSystemPromptPreset(append string) QueryOption {
	return func(c *queryConfig) { c.systemPromptPreset = append }
}

// WithSystemPromptFile loads the system prompt from a file path.
func WithSystemPromptFile(path string) QueryOption {
	return func(c *queryConfig) { c.systemPromptFile = path }
}

// --- Conversation ---

// WithResume resumes an existing session by ID.
func WithResume(sessionID string) QueryOption {
	return func(c *queryConfig) { c.resumeSessionID = sessionID }
}

// WithContinue continues the most recent session.
func WithContinue() QueryOption {
	return func(c *queryConfig) { c.continueSession = true }
}

// WithSessionID sets a specific session ID for the conversation.
func WithSessionID(id string) QueryOption {
	return func(c *queryConfig) { c.sessionID = id }
}

// WithForkSession forks the resumed session instead of continuing in-place.
func WithForkSession() QueryOption {
	return func(c *queryConfig) { c.forkSession = true }
}

// WithMaxTurns limits the number of agentic turns.
func WithMaxTurns(n int) QueryOption {
	return func(c *queryConfig) { c.maxTurns = n }
}

// WithMaxBudgetUSD sets a maximum spend limit in USD.
func WithMaxBudgetUSD(budget float64) QueryOption {
	return func(c *queryConfig) { c.maxBudgetUSD = budget }
}

// --- Environment ---

// WithCwd sets the working directory for the claude process.
func WithCwd(dir string) QueryOption {
	return func(c *queryConfig) { c.cwd = dir }
}

// WithEnv sets additional environment variables for the claude process.
func WithEnv(env map[string]string) QueryOption {
	return func(c *queryConfig) { c.env = env }
}

// WithAdditionalDirectories adds directories to the file search scope.
func WithAdditionalDirectories(dirs ...string) QueryOption {
	return func(c *queryConfig) { c.additionalDirs = dirs }
}

// WithCLIPath sets an explicit path to the claude CLI binary.
func WithCLIPath(path string) QueryOption {
	return func(c *queryConfig) { c.cliPath = path }
}

// WithAPIKey sets the Anthropic API key for the claude process.
// This is passed via the ANTHROPIC_API_KEY environment variable.
func WithAPIKey(key string) QueryOption {
	return func(c *queryConfig) { c.apiKey = key }
}

// WithSettingSources adds Claude Code settings files.
func WithSettingSources(sources ...SettingSource) QueryOption {
	return func(c *queryConfig) { c.settingSources = sources }
}

// --- Extensions ---

// WithMcpServers configures MCP servers for the session.
func WithMcpServers(servers map[string]McpServerConfig) QueryOption {
	return func(c *queryConfig) { c.mcpServers = servers }
}

// WithAgents configures subagent definitions.
func WithAgents(agents map[string]AgentDefinition) QueryOption {
	return func(c *queryConfig) { c.agents = agents }
}

// WithHooks configures hook callbacks for agent lifecycle events.
func WithHooks(hooks map[HookEvent][]HookMatcher) QueryOption {
	return func(c *queryConfig) { c.hooks = hooks }
}

// WithPlugins configures plugins to load.
func WithPlugins(plugins ...PluginConfig) QueryOption {
	return func(c *queryConfig) { c.plugins = plugins }
}

// --- Output ---

// WithOutputFormat sets a JSON schema for structured output.
func WithOutputFormat(schema map[string]any) QueryOption {
	return func(c *queryConfig) { c.outputFormat = schema }
}

// WithIncludePartialMessages includes streaming partial messages in the output.
func WithIncludePartialMessages() QueryOption {
	return func(c *queryConfig) { c.includePartialMessages = true }
}

// --- Tool Config ---

// WithToolConfig sets tool-specific configuration (e.g., AskUserQuestion preview format).
func WithToolConfig(config ToolConfig) QueryOption {
	return func(c *queryConfig) { c.toolConfig = &config }
}

// --- Debugging ---

// WithStderr provides a callback for claude process stderr output.
func WithStderr(fn func(string)) QueryOption {
	return func(c *queryConfig) { c.stderrFunc = fn }
}

// WithDebug enables debug mode.
func WithDebug() QueryOption {
	return func(c *queryConfig) { c.debug = true }
}

// WithDebugFile writes debug output to a file.
func WithDebugFile(path string) QueryOption {
	return func(c *queryConfig) { c.debugFile = path }
}

// WithMaxBufferSize sets the maximum stdout buffer size in bytes.
func WithMaxBufferSize(size int) QueryOption {
	return func(c *queryConfig) { c.maxBufferSize = size }
}

// --- Session Persistence ---

// WithPersistSession controls whether the session is persisted to disk.
// Default is true. Set to false for in-memory-only sessions.
func WithPersistSession(persist bool) QueryOption {
	return func(c *queryConfig) { c.persistSession = &persist }
}

// --- Beta Features ---

// WithBetas enables beta feature flags (e.g., SdkBetaContext1M).
func WithBetas(betas ...SdkBeta) QueryOption {
	return func(c *queryConfig) { c.betas = betas }
}

// --- Budget ---

// WithTaskBudget sets token budgets for background tasks.
func WithTaskBudget(budget TaskBudget) QueryOption {
	return func(c *queryConfig) { c.taskBudget = &budget }
}

// --- Sandbox ---

// WithSandbox configures filesystem and network isolation.
func WithSandbox(settings SandboxSettings) QueryOption {
	return func(c *queryConfig) { c.sandbox = &settings }
}

// --- Identity ---

// WithUser sets the user identifier for the session.
func WithUser(user string) QueryOption {
	return func(c *queryConfig) { c.user = user }
}

// --- File Checkpointing ---

// WithFileCheckpointing enables file change tracking for rewind support.
func WithFileCheckpointing() QueryOption {
	return func(c *queryConfig) { c.fileCheckpointing = true }
}

// --- Timeouts ---

// WithProcessTimeout sets the maximum duration for the claude process.
func WithProcessTimeout(d time.Duration) QueryOption {
	return func(c *queryConfig) { c.processTimeout = d }
}

// --- Extra CLI Arguments ---

// WithExtraArgs passes additional CLI arguments directly to the claude process.
// This is an escape hatch for features not yet covered by typed options.
func WithExtraArgs(args ...string) QueryOption {
	return func(c *queryConfig) { c.extraArgs = args }
}

// --- Agent ---

// WithAgent sets the main thread agent name, applying the agent's system prompt,
// tool restrictions, and model to the main conversation.
// Equivalent to the --agent CLI flag.
func WithAgent(name string) QueryOption {
	return func(c *queryConfig) { c.agent = name }
}

// --- Elicitation ---

// WithOnElicitation sets a callback for handling MCP elicitation requests.
// Called when an MCP server requests user input (form fields, URL auth, etc.).
// If not set, elicitation requests are automatically declined.
func WithOnElicitation(fn OnElicitationFunc) QueryOption {
	return func(c *queryConfig) { c.onElicitation = fn }
}

// --- Prompt Suggestions ---

// WithPromptSuggestions enables AI-generated prompt suggestions after each turn.
// When enabled, the agent emits a PromptSuggestionMessage after each result.
func WithPromptSuggestions(enabled bool) QueryOption {
	return func(c *queryConfig) { c.promptSuggestions = &enabled }
}

// WithAgentProgressSummaries enables periodic AI-generated progress summaries
// for running subagents, emitted on task_progress events via the Summary field.
func WithAgentProgressSummaries(enabled bool) QueryOption {
	return func(c *queryConfig) { c.agentProgressSummaries = &enabled }
}

// --- Hook Events ---

// WithIncludeHookEvents includes hook lifecycle events (hook_started, hook_progress,
// hook_response) in the output stream. SessionStart and Setup hooks are always emitted.
func WithIncludeHookEvents(enabled bool) QueryOption {
	return func(c *queryConfig) { c.includeHookEvents = enabled }
}

// --- Resume ---

// WithResumeSessionAt resumes a session only up to the specified message UUID.
// Use with WithResume to resume from a specific point in the conversation.
func WithResumeSessionAt(messageUUID string) QueryOption {
	return func(c *queryConfig) { c.resumeSessionAt = messageUUID }
}

// --- Settings ---

// WithSettings provides inline settings or a path to a settings file.
// Accepts map[string]any (inline settings object) or string (file path).
// Inline settings are serialized to a temp file. Settings are applied at
// the highest priority "flag settings" layer.
func WithSettings(settings any) QueryOption {
	return func(c *queryConfig) { c.settings = settings }
}

// --- MCP Config ---

// WithStrictMcpConfig enforces strict validation of MCP server configurations.
// When enabled, invalid configurations cause errors instead of warnings.
func WithStrictMcpConfig(enabled bool) QueryOption {
	return func(c *queryConfig) { c.strictMcpConfig = enabled }
}

// applyDefaults fills in default values for unset fields.
func (c *queryConfig) applyDefaults() {
	if c.maxBufferSize <= 0 {
		c.maxBufferSize = 1 << 20 // 1MB
	}
	if c.processTimeout <= 0 {
		c.processTimeout = 30 * time.Minute
	}
}
