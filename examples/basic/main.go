package main

import (
	"context"
	"fmt"
	"os"

	agentsdk "github.com/agentserver/claude-agent-sdk-go"
)

func main() {
	ctx := context.Background()

	stream := agentsdk.Query(ctx, "What is 2+2? Reply with just the number.",
		agentsdk.WithMaxTurns(1),
	)
	defer stream.Close()

	for stream.Next() {
		msg := stream.Current()

		switch msg.Type {
		case "assistant":
			assistant, _ := msg.AsAssistant()
			for _, block := range assistant.Message.Content {
				if text, ok := block.AsText(); ok {
					fmt.Print(text.Text)
				}
			}
		case "result":
			result, _ := msg.AsResult()
			fmt.Printf("\n\n--- Done (turns: %d, cost: $%.4f) ---\n",
				result.NumTurns,
				result.TotalCostUSD,
			)
		}
	}

	if err := stream.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
