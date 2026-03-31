package main

import (
	"context"
	"fmt"
	"os"
	"regexp"

	agentsdk "github.com/anthropics/claude-agent-sdk-go"
)

func main() {
	ctx := context.Background()

	// Define hooks: log tool uses, auto-approve safe tools, block dangerous ones.
	hooks := map[agentsdk.HookEvent][]agentsdk.HookMatcher{
		agentsdk.HookPreToolUse: {
			{
				// Auto-approve read-only tools.
				Matcher: "Read|Glob|Grep",
				Hooks: []agentsdk.HookCallback{
					func(ctx context.Context, input agentsdk.HookInput, toolUseID string) (agentsdk.HookOutput, error) {
						fmt.Printf("[Hook] Auto-approving tool: %s\n", input.ToolName)
						return agentsdk.HookOutput{
							HookSpecificOutput: &agentsdk.HookSpecificOutput{
								HookEventName:      "PreToolUse",
								PermissionDecision: "allow",
							},
						}, nil
					},
				},
			},
			{
				// Block Bash commands containing "rm".
				Matcher: "Bash",
				Hooks: []agentsdk.HookCallback{
					func(ctx context.Context, input agentsdk.HookInput, toolUseID string) (agentsdk.HookOutput, error) {
						if cmd, ok := input.ToolInput["command"].(string); ok {
							if matched, _ := regexp.MatchString(`\brm\b`, cmd); matched {
								fmt.Printf("[Hook] BLOCKED dangerous command: %s\n", cmd)
								return agentsdk.HookOutput{
									Decision: "block",
									Reason:   "rm commands are not allowed",
								}, nil
							}
						}
						return agentsdk.HookOutput{
							HookSpecificOutput: &agentsdk.HookSpecificOutput{
								HookEventName:      "PreToolUse",
								PermissionDecision: "allow",
							},
						}, nil
					},
				},
			},
		},
		agentsdk.HookPostToolUse: {
			{
				Hooks: []agentsdk.HookCallback{
					func(ctx context.Context, input agentsdk.HookInput, toolUseID string) (agentsdk.HookOutput, error) {
						fmt.Printf("[Hook] Tool completed: %s\n", input.ToolName)
						return agentsdk.HookOutput{Continue: true}, nil
					},
				},
			},
		},
	}

	stream := agentsdk.Query(ctx, "List the Go files in the current directory",
		agentsdk.WithMaxTurns(5),
		agentsdk.WithHooks(hooks),
	)
	defer stream.Close()

	for stream.Next() {
		msg := stream.Current()

		switch msg.Type {
		case "assistant":
			assistant, _ := msg.AsAssistant()
			for _, block := range assistant.Content {
				if text, ok := block.AsText(); ok {
					fmt.Print(text.Text)
				}
				if toolUse, ok := block.AsToolUse(); ok {
					fmt.Printf("\n[Tool: %s]\n", toolUse.Name)
				}
			}
		case "result":
			result, _ := msg.AsResult()
			fmt.Printf("\n\n--- Done (turns: %d) ---\n", result.NumTurns)
		}
	}

	if err := stream.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
