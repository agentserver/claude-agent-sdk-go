package agentsdk

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/anthropics/claude-agent-sdk-go/internal/clilookup"
	"github.com/anthropics/claude-agent-sdk-go/internal/transport"
)

// Stream iterates over SDK messages from the claude process.
// Usage:
//
//	stream := agentsdk.Query(ctx, "Hello", opts...)
//	defer stream.Close()
//	for stream.Next() {
//	    msg := stream.Current()
//	    // handle msg
//	}
//	if err := stream.Err(); err != nil {
//	    // handle error
//	}
type Stream struct {
	transport      transport.Transport
	controlHandler *controlHandler
	ctx            context.Context
	msgCh          <-chan json.RawMessage
	current        SDKMessage
	err            error
	result         *ResultMessage
	closed         bool
	oneShot        bool // true for Query() — closes stdin after result
}

// Next advances to the next message. Returns false when the stream is
// exhausted or an error occurred. Always check Err() after Next() returns false.
// Control protocol messages (hook callbacks, permission checks, MCP tool calls)
// are handled internally and not exposed to the caller.
func (s *Stream) Next() bool {
	for {
		if s.err != nil || s.closed {
			return false
		}

		raw, ok := <-s.msgCh
		if !ok {
			// Channel closed — process exited.
			if s.transport != nil {
				if exitErr := s.transport.ExitError(); exitErr != nil && s.result == nil {
					s.err = &ProcessError{ExitCode: -1, Stderr: exitErr.Error()}
				}
			}
			return false
		}

		// Route control requests through the handler (hooks, permissions, MCP tools).
		if s.controlHandler != nil && s.controlHandler.handleMessage(s.ctx, raw) {
			continue // Control request handled, get next message.
		}

		var msg SDKMessage
		if err := json.Unmarshal(raw, &msg); err != nil {
			s.err = &MessageParseError{Data: raw, Err: err}
			return false
		}

		s.current = msg

		// Capture result message for Result() convenience method.
		if result, ok := msg.AsResult(); ok {
			s.result = result
			if s.oneShot && s.transport != nil {
				s.transport.EndInput()
			}
		}

		return true
	}
}

// Current returns the current message. Valid only after Next() returns true.
func (s *Stream) Current() SDKMessage {
	return s.current
}

// Err returns the error that stopped iteration, or nil for clean finish.
func (s *Stream) Err() error {
	return s.err
}

// Close shuts down the claude process and releases resources.
func (s *Stream) Close() error {
	if s.closed {
		return nil
	}
	s.closed = true
	if s.transport != nil {
		return s.transport.Close()
	}
	return nil
}

// Interrupt sends an interrupt signal to the claude process.
func (s *Stream) Interrupt() error {
	req := map[string]any{
		"type":       "control_request",
		"request_id": "sdk_interrupt",
		"request": map[string]any{
			"subtype": "interrupt",
		},
	}
	data, err := json.Marshal(req)
	if err != nil {
		return err
	}
	return s.transport.Write(string(data))
}

// Send writes a user message to the stream for multi-turn conversations.
// This is equivalent to TS SDK's Query.streamInput().
// The stream must not have been closed or ended.
func (s *Stream) Send(prompt string) error {
	if s.closed || s.transport == nil {
		return fmt.Errorf("stream closed")
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
	return s.transport.Write(string(data))
}

// Result drains the stream and returns the final ResultMessage.
// This is a convenience that consumes all remaining messages.
func (s *Stream) Result() (*ResultMessage, error) {
	for s.Next() {
		// Just consume.
	}
	if s.err != nil {
		return nil, s.err
	}
	if s.result == nil {
		return nil, fmt.Errorf("no result message received")
	}
	return s.result, nil
}

// NewStreamFromTransport creates a Stream from an existing transport.
// This is primarily used for testing with mock transports.
func NewStreamFromTransport(tp transport.Transport) *Stream {
	return &Stream{
		transport: tp,
		ctx:       context.Background(),
		msgCh:     tp.ReadMessages(),
	}
}

// TestQueryConfig exposes queryConfig fields for external tests.
type TestQueryConfig struct {
	Model          string
	MaxTurns       int
	PermissionMode PermissionMode
	AllowedTools   []string
}

// BuildCLIArgsForTest converts a TestQueryConfig to CLI arguments for testing.
func BuildCLIArgsForTest(tc TestQueryConfig) []string {
	cfg := &queryConfig{
		model:          tc.Model,
		maxTurns:       tc.MaxTurns,
		permissionMode: tc.PermissionMode,
		allowedTools:   tc.AllowedTools,
	}
	return buildCLIArgs(cfg)
}

// Query creates a new Claude Code session and streams messages.
// It spawns a claude subprocess, sends the prompt, and returns
// a Stream that yields messages as they arrive.
func Query(ctx context.Context, prompt string, opts ...QueryOption) *Stream {
	cfg := &queryConfig{}
	for _, opt := range opts {
		opt(cfg)
	}
	cfg.applyDefaults()

	// Find CLI binary.
	cliPath, err := clilookup.FindCLI(cfg.cliPath)
	if err != nil {
		// Wrap internal error in public CLINotFoundError.
		if nf, ok := err.(*clilookup.NotFoundError); ok {
			return &Stream{err: &CLINotFoundError{SearchedPaths: nf.SearchedPaths}}
		}
		return &Stream{err: err}
	}

	// Build CLI arguments from config.
	args := buildCLIArgs(cfg)

	// Merge API key into env if set.
	env := cfg.env
	if cfg.apiKey != "" {
		if env == nil {
			env = make(map[string]string)
		}
		env["ANTHROPIC_API_KEY"] = cfg.apiKey
	}

	// Create transport.
	tp := transport.NewSubprocessTransport(transport.SubprocessConfig{
		CLIPath:       cliPath,
		Args:          args,
		Cwd:           cfg.cwd,
		Env:           env,
		MaxBufferSize: cfg.maxBufferSize,
		StderrFunc:    cfg.stderrFunc,
	})

	// Connect (start process).
	if err := tp.Connect(ctx); err != nil {
		return &Stream{err: err}
	}

	// Create control handler for hooks, permissions, and MCP tool calls.
	handler := newControlHandler(cfg, tp)

	// Send initialize handshake BEFORE user message so hooks/agents/schema
	// are registered with the CLI before processing begins.
	if err := handler.sendInitialize(ctx); err != nil {
		tp.Close()
		return &Stream{err: fmt.Errorf("initialize: %w", err)}
	}

	// Send prompt via stdin as a user message.
	userMsg := map[string]any{
		"type": "user",
		"message": map[string]any{
			"role":    "user",
			"content": prompt,
		},
	}
	data, err := json.Marshal(userMsg)
	if err != nil {
		tp.Close()
		return &Stream{err: fmt.Errorf("marshal prompt: %w", err)}
	}
	if err := tp.Write(string(data)); err != nil {
		tp.Close()
		return &Stream{err: fmt.Errorf("write prompt: %w", err)}
	}

	return &Stream{
		transport:      tp,
		controlHandler: handler,
		ctx:            ctx,
		msgCh:          tp.ReadMessages(),
		oneShot:        true,
	}
}

// buildCLIArgs converts queryConfig to CLI arguments.
func buildCLIArgs(cfg *queryConfig) []string {
	var args []string

	if cfg.model != "" {
		args = append(args, "--model", cfg.model)
	}
	if cfg.permissionMode != "" {
		args = append(args, "--permission-mode", string(cfg.permissionMode))
	}
	if cfg.bypassPermissions && cfg.allowDangerouslySkipPermissions {
		args = append(args, "--dangerously-skip-permissions")
	}
	if cfg.maxTurns > 0 {
		args = append(args, "--max-turns", fmt.Sprintf("%d", cfg.maxTurns))
	}
	for _, tool := range cfg.allowedTools {
		args = append(args, "--allowedTools", tool)
	}
	for _, tool := range cfg.tools {
		args = append(args, "--tools", tool)
	}
	if cfg.toolsPreset {
		args = append(args, "--tools-preset", "claude_code")
	}
	for _, tool := range cfg.disallowedTools {
		args = append(args, "--disallowedTools", tool)
	}

	// MCP servers (passed as JSON config — excludes SDK servers which are handled in-process).
	if len(cfg.mcpServers) > 0 {
		cliServers := make(map[string]McpServerConfig)
		for name, srv := range cfg.mcpServers {
			if srv.SDK == nil {
				cliServers[name] = srv
			}
		}
		if len(cliServers) > 0 {
			mcpJSON, err := json.Marshal(cliServers)
			if err == nil {
				args = append(args, "--mcp-config", string(mcpJSON))
			}
		}
	}

	if len(cfg.agents) > 0 {
		agentsJSON, err := json.Marshal(cfg.agents)
		if err == nil {
			args = append(args, "--agents", string(agentsJSON))
		}
	}

	if cfg.systemPrompt != "" {
		args = append(args, "--system-prompt", cfg.systemPrompt)
	}
	if cfg.systemPromptPreset != "" {
		args = append(args, "--system-prompt-preset", "claude_code")
		args = append(args, "--append-system-prompt", cfg.systemPromptPreset)
	}
	if cfg.systemPromptFile != "" {
		args = append(args, "--system-prompt-file", cfg.systemPromptFile)
	}

	if cfg.agent != "" {
		args = append(args, "--agent", cfg.agent)
	}
	if cfg.resumeSessionID != "" {
		args = append(args, "--resume", cfg.resumeSessionID)
	}
	if cfg.resumeSessionAt != "" {
		args = append(args, "--resume-session-at", cfg.resumeSessionAt)
	}
	if cfg.continueSession {
		args = append(args, "--continue")
	}
	if cfg.forkSession {
		args = append(args, "--fork-session")
	}
	if cfg.sessionID != "" {
		args = append(args, "--session-id", cfg.sessionID)
	}
	if cfg.includePartialMessages {
		args = append(args, "--include-partial-messages")
	}
	if cfg.includeHookEvents {
		args = append(args, "--include-hook-events")
	}
	if cfg.strictMcpConfig {
		args = append(args, "--strict-mcp-config")
	}
	if cfg.debug {
		args = append(args, "--debug")
	}
	if cfg.debugFile != "" {
		args = append(args, "--debug-file", cfg.debugFile)
	}
	for _, dir := range cfg.additionalDirs {
		args = append(args, "--add-dir", dir)
	}
	for _, source := range cfg.settingSources {
		args = append(args, "--setting-source", string(source))
	}
	for _, plugin := range cfg.plugins {
		if plugin.Path != "" {
			args = append(args, "--plugin", plugin.Path)
		}
	}
	if cfg.persistSession != nil && !*cfg.persistSession {
		args = append(args, "--no-persist-session")
	}
	if cfg.fileCheckpointing {
		args = append(args, "--enable-file-checkpointing")
	}
	for _, beta := range cfg.betas {
		args = append(args, "--beta", string(beta))
	}
	if cfg.effort != "" {
		args = append(args, "--effort", string(cfg.effort))
	}
	if cfg.thinking != nil && cfg.thinking.Type == "enabled" && cfg.thinking.BudgetTokens > 0 {
		args = append(args, "--max-thinking-tokens", fmt.Sprintf("%d", cfg.thinking.BudgetTokens))
	} else if cfg.maxThinkingTokens != nil {
		args = append(args, "--max-thinking-tokens", fmt.Sprintf("%d", *cfg.maxThinkingTokens))
	}
	if cfg.outputFormat != nil {
		formatJSON, err := json.Marshal(cfg.outputFormat)
		if err == nil {
			args = append(args, "--output-format-json", string(formatJSON))
		}
	}
	if cfg.maxBudgetUSD > 0 {
		args = append(args, "--max-budget-usd", fmt.Sprintf("%.2f", cfg.maxBudgetUSD))
	}
	if cfg.fallbackModel != "" {
		args = append(args, "--fallback-model", cfg.fallbackModel)
	}
	if cfg.taskBudget != nil && cfg.taskBudget.Total > 0 {
		args = append(args, "--task-budget", fmt.Sprintf("%d", cfg.taskBudget.Total))
	}
	if cfg.user != "" {
		args = append(args, "--user", cfg.user)
	}
	if cfg.sandbox != nil {
		sandboxJSON, err := json.Marshal(cfg.sandbox)
		if err == nil {
			args = append(args, "--sandbox-config", string(sandboxJSON))
		}
	}
	if cfg.permissionPromptToolName != "" {
		args = append(args, "--permission-prompt-tool-name", cfg.permissionPromptToolName)
	}
	if cfg.toolConfig != nil {
		toolConfigJSON, err := json.Marshal(cfg.toolConfig)
		if err == nil {
			args = append(args, "--tool-config", string(toolConfigJSON))
		}
	}
	if cfg.settings != nil {
		switch v := cfg.settings.(type) {
		case string:
			args = append(args, "--settings", v)
		default:
			// Inline settings object — serialize to a temp file.
			data, err := json.Marshal(v)
			if err == nil {
				tmpDir := os.TempDir()
				tmpFile := filepath.Join(tmpDir, fmt.Sprintf("claude-sdk-settings-%d.json", os.Getpid()))
				if os.WriteFile(tmpFile, data, 0600) == nil {
					args = append(args, "--settings", tmpFile)
				}
			}
		}
	}

	args = append(args, cfg.extraArgs...)

	return args
}
