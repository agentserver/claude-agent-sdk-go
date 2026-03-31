package main

import (
	"context"
	"fmt"
	"os"

	agentsdk "github.com/anthropics/claude-agent-sdk-go"
)

type AddInput struct {
	A float64 `json:"a"`
	B float64 `json:"b"`
}

func main() {
	ctx := context.Background()

	// Define an in-process MCP server with a simple "add" tool.
	addTool := agentsdk.Tool("add", "Add two numbers together", func(ctx context.Context, input AddInput) (*agentsdk.McpToolResult, error) {
		result := input.A + input.B
		return &agentsdk.McpToolResult{
			Content: []agentsdk.McpToolContent{
				{Type: "text", Text: fmt.Sprintf("%g", result)},
			},
		}, nil
	})

	sdkServer := agentsdk.CreateSdkMcpServer("math-tools", "1.0.0", addTool)

	stream := agentsdk.Query(ctx, "What is 123.45 + 678.90? Use the add tool.",
		agentsdk.WithMaxTurns(3),
		agentsdk.WithMcpServers(map[string]agentsdk.McpServerConfig{
			"math": {SDK: sdkServer},
		}),
	)
	defer stream.Close()

	for stream.Next() {
		msg := stream.Current()
		if assistant, ok := msg.AsAssistant(); ok {
			for _, block := range assistant.Content {
				if text, ok := block.AsText(); ok {
					fmt.Print(text.Text)
				}
			}
		}
	}

	if err := stream.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
