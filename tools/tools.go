// Package tools provides typed Go structs for all Claude Code built-in tool
// inputs and outputs. These correspond to the TypeScript SDK's sdk-tools
// export (@anthropic-ai/claude-agent-sdk/sdk-tools).
//
// Use these types to inspect or process tool_use blocks in AssistantMessage
// content and tool results in ToolResultMessage / PostToolUse hooks.
package tools

// StructuredPatch represents a diff hunk in file edit/write outputs.
type StructuredPatch struct {
	OldStart int      `json:"oldStart"`
	OldLines int      `json:"oldLines"`
	NewStart int      `json:"newStart"`
	NewLines int      `json:"newLines"`
	Lines    []string `json:"lines"`
}

// GitDiff represents a git diff for a single file.
type GitDiff struct {
	Filename   string  `json:"filename"`
	Status     string  `json:"status"` // "modified", "added"
	Additions  int     `json:"additions"`
	Deletions  int     `json:"deletions"`
	Changes    int     `json:"changes"`
	Patch      string  `json:"patch"`
	Repository *string `json:"repository,omitempty"`
}

// Usage tracks token usage for agent tool results.
type Usage struct {
	InputTokens              int  `json:"input_tokens"`
	OutputTokens             int  `json:"output_tokens"`
	CacheCreationInputTokens *int `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     *int `json:"cache_read_input_tokens"`
	ServerToolUse            *struct {
		WebSearchRequests int `json:"web_search_requests"`
		WebFetchRequests  int `json:"web_fetch_requests"`
	} `json:"server_tool_use"`
	ServiceTier   *string `json:"service_tier"`
	CacheCreation *struct {
		Ephemeral1hInputTokens int `json:"ephemeral_1h_input_tokens"`
		Ephemeral5mInputTokens int `json:"ephemeral_5m_input_tokens"`
	} `json:"cache_creation"`
}

// TextContent is a simple text content block.
type TextContent struct {
	Type string `json:"type"` // "text"
	Text string `json:"text"`
}
