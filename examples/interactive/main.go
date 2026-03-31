package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	agentsdk "github.com/anthropics/claude-agent-sdk-go"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client := agentsdk.NewClient(
		agentsdk.WithMaxTurns(10),
		agentsdk.WithPermissionMode(agentsdk.PermissionBypassAll),
		agentsdk.WithAllowDangerouslySkipPermissions(),
	)

	if err := client.Connect(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Connect error: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	// Print messages in background.
	go func() {
		for msg := range client.Messages() {
			switch msg.Type {
			case "assistant":
				assistant, _ := msg.AsAssistant()
				for _, block := range assistant.Content {
					if text, ok := block.AsText(); ok {
						fmt.Print(text.Text)
					}
				}
			case "result":
				fmt.Println("\n--- Turn complete ---")
			}
		}
	}()

	// Read user input.
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("\nYou: ")
	for scanner.Scan() {
		text := strings.TrimSpace(scanner.Text())
		if text == "" {
			fmt.Print("You: ")
			continue
		}
		if text == "/quit" {
			break
		}

		if err := client.Send(ctx, text); err != nil {
			fmt.Fprintf(os.Stderr, "Send error: %v\n", err)
			break
		}
		fmt.Print("\nYou: ")
	}
}
