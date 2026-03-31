package agentsdk

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/anthropics/claude-agent-sdk-go/internal/clilookup"
)

// SessionInfo contains metadata about a claude session.
// Timestamps are milliseconds since epoch, matching the TS/Python SDK conventions.
type SessionInfo struct {
	ID           string `json:"id"`
	Summary      string `json:"summary,omitempty"`
	LastModified int64  `json:"last_modified"`
	FileSize     int64  `json:"file_size,omitempty"`
	CustomTitle  string `json:"custom_title,omitempty"`
	FirstPrompt  string `json:"first_prompt,omitempty"`
	GitBranch    string `json:"git_branch,omitempty"`
	Cwd          string `json:"cwd,omitempty"`
	Tag          string `json:"tag,omitempty"`
	CreatedAt    *int64 `json:"created_at,omitempty"`
}

// SessionMessage is a single message within a session.
type SessionMessage struct {
	Role    string `json:"role"` // "user", "assistant", "system"
	Content any    `json:"content"`
}

// ListSessionsOptions configures the list sessions command.
type ListSessionsOptions struct {
	CLIPath          string
	Cwd              string
	Limit            int
	Offset           int
	IncludeWorktrees bool
}

// GetSessionInfoOptions configures the get session info command.
type GetSessionInfoOptions struct {
	CLIPath string
	Cwd     string
}

// GetSessionMessagesOptions configures the get session messages command.
type GetSessionMessagesOptions struct {
	CLIPath               string
	Cwd                   string
	Limit                 int
	Offset                int
	IncludeSystemMessages bool
	IncludeHookEvents     bool
}

// SessionMutationOptions configures session mutation commands.
type SessionMutationOptions struct {
	CLIPath string
	Cwd     string
}

// ListSessions returns all available sessions.
func ListSessions(opts *ListSessionsOptions) ([]SessionInfo, error) {
	var explicitPath string
	if opts != nil {
		explicitPath = opts.CLIPath
	}
	cliPath, err := findCLIForSession(explicitPath)
	if err != nil {
		return nil, err
	}

	args := []string{"sessions", "list", "--output-format", "json"}
	if opts != nil {
		if opts.Limit > 0 {
			args = append(args, "--limit", fmt.Sprintf("%d", opts.Limit))
		}
		if opts.Offset > 0 {
			args = append(args, "--offset", fmt.Sprintf("%d", opts.Offset))
		}
		if opts.IncludeWorktrees {
			args = append(args, "--include-worktrees")
		}
	}

	out, err := exec.Command(cliPath, args...).Output()
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}

	var sessions []SessionInfo
	if err := json.Unmarshal(out, &sessions); err != nil {
		return nil, fmt.Errorf("parse sessions: %w", err)
	}
	return sessions, nil
}

// GetSessionInfo returns metadata for a specific session.
func GetSessionInfo(sessionID string, opts *GetSessionInfoOptions) (*SessionInfo, error) {
	var explicitPath string
	if opts != nil {
		explicitPath = opts.CLIPath
	}
	cliPath, err := findCLIForSession(explicitPath)
	if err != nil {
		return nil, err
	}

	out, err := exec.Command(cliPath, "sessions", "get", sessionID, "--output-format", "json").Output()
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}

	var info SessionInfo
	if err := json.Unmarshal(out, &info); err != nil {
		return nil, fmt.Errorf("parse session: %w", err)
	}
	return &info, nil
}

// GetSessionMessages returns all messages in a session.
func GetSessionMessages(sessionID string, opts *GetSessionMessagesOptions) ([]SessionMessage, error) {
	var explicitPath string
	if opts != nil {
		explicitPath = opts.CLIPath
	}
	cliPath, err := findCLIForSession(explicitPath)
	if err != nil {
		return nil, err
	}

	args := []string{"sessions", "messages", sessionID, "--output-format", "json"}
	if opts != nil {
		if opts.Limit > 0 {
			args = append(args, "--limit", fmt.Sprintf("%d", opts.Limit))
		}
		if opts.Offset > 0 {
			args = append(args, "--offset", fmt.Sprintf("%d", opts.Offset))
		}
		if opts.IncludeSystemMessages {
			args = append(args, "--include-system-messages")
		}
		if opts.IncludeHookEvents {
			args = append(args, "--include-hook-events")
		}
	}

	out, err := exec.Command(cliPath, args...).Output()
	if err != nil {
		return nil, fmt.Errorf("get session messages: %w", err)
	}

	var messages []SessionMessage
	if err := json.Unmarshal(out, &messages); err != nil {
		return nil, fmt.Errorf("parse messages: %w", err)
	}
	return messages, nil
}

// RenameSession renames a session.
func RenameSession(sessionID, title string, opts *SessionMutationOptions) error {
	var explicitPath string
	if opts != nil {
		explicitPath = opts.CLIPath
	}
	cliPath, err := findCLIForSession(explicitPath)
	if err != nil {
		return err
	}

	if out, err := exec.Command(cliPath, "sessions", "rename", sessionID, title).CombinedOutput(); err != nil {
		return fmt.Errorf("rename session: %s: %w", strings.TrimSpace(string(out)), err)
	}
	return nil
}

// TagSession sets or clears the tag on a session.
func TagSession(sessionID string, tag *string, opts *SessionMutationOptions) error {
	var explicitPath string
	if opts != nil {
		explicitPath = opts.CLIPath
	}
	cliPath, err := findCLIForSession(explicitPath)
	if err != nil {
		return err
	}

	args := []string{"sessions", "tag", sessionID}
	if tag != nil {
		args = append(args, *tag)
	} else {
		args = append(args, "--clear")
	}

	if out, err := exec.Command(cliPath, args...).CombinedOutput(); err != nil {
		return fmt.Errorf("tag session: %s: %w", strings.TrimSpace(string(out)), err)
	}
	return nil
}

// DeleteSession deletes a session.
func DeleteSession(sessionID string, opts *SessionMutationOptions) error {
	var explicitPath string
	if opts != nil {
		explicitPath = opts.CLIPath
	}
	cliPath, err := findCLIForSession(explicitPath)
	if err != nil {
		return err
	}

	if out, err := exec.Command(cliPath, "sessions", "delete", sessionID).CombinedOutput(); err != nil {
		return fmt.Errorf("delete session: %s: %w", strings.TrimSpace(string(out)), err)
	}
	return nil
}

// ForkSessionResult contains the result of forking a session.
type ForkSessionResult struct {
	SessionID string `json:"session_id"`
}

// ForkSession creates a new session by forking an existing one.
func ForkSession(sessionID string, opts *SessionMutationOptions, upToMessageID, title string) (*ForkSessionResult, error) {
	var explicitPath string
	if opts != nil {
		explicitPath = opts.CLIPath
	}
	cliPath, err := findCLIForSession(explicitPath)
	if err != nil {
		return nil, err
	}

	args := []string{"sessions", "fork", sessionID, "--output-format", "json"}
	if upToMessageID != "" {
		args = append(args, "--up-to", upToMessageID)
	}
	if title != "" {
		args = append(args, "--title", title)
	}

	out, err := exec.Command(cliPath, args...).Output()
	if err != nil {
		return nil, fmt.Errorf("fork session: %w", err)
	}

	var result ForkSessionResult
	if err := json.Unmarshal(out, &result); err != nil {
		return nil, fmt.Errorf("parse fork result: %w", err)
	}
	return &result, nil
}

func findCLIForSession(cliPath string) (string, error) {
	return clilookup.FindCLI(cliPath)
}
