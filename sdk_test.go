package agentsdk_test

import (
	"encoding/json"
	"testing"

	agentsdk "github.com/anthropics/claude-agent-sdk-go"
	"github.com/anthropics/claude-agent-sdk-go/internal/transport"
)

func TestStream_SingleAssistantMessage(t *testing.T) {
	msg := json.RawMessage(`{"type":"assistant","content":[{"type":"text","text":"Hello!"}],"model":"claude-sonnet-4-6","session_id":"s1","uuid":"u1"}`)
	result := json.RawMessage(`{"type":"result","subtype":"success","result":"Hello!","session_id":"s1","is_error":false,"num_turns":1}`)

	mock := transport.NewMockTransport(msg, result)
	mock.Connect(nil)

	stream := agentsdk.NewStreamFromTransport(mock)
	defer stream.Close()

	// First message: assistant.
	if !stream.Next() {
		t.Fatal("expected Next() to return true for assistant message")
	}
	cur := stream.Current()
	if cur.Type != "assistant" {
		t.Errorf("expected type assistant, got %s", cur.Type)
	}
	assistant, ok := cur.AsAssistant()
	if !ok {
		t.Fatal("AsAssistant returned false")
	}
	if len(assistant.Content) == 0 {
		t.Fatal("expected content blocks")
	}
	text, ok := assistant.Content[0].AsText()
	if !ok {
		t.Fatal("expected text block")
	}
	if text.Text != "Hello!" {
		t.Errorf("expected 'Hello!', got %q", text.Text)
	}

	// Second message: result.
	if !stream.Next() {
		t.Fatal("expected Next() to return true for result message")
	}
	if stream.Current().Type != "result" {
		t.Errorf("expected type result, got %s", stream.Current().Type)
	}

	// Stream exhausted.
	if stream.Next() {
		t.Fatal("expected Next() to return false after stream ends")
	}
	if stream.Err() != nil {
		t.Fatalf("unexpected error: %v", stream.Err())
	}
}

func TestStream_Result(t *testing.T) {
	msg := json.RawMessage(`{"type":"assistant","content":[{"type":"text","text":"Hi"}],"session_id":"s1"}`)
	result := json.RawMessage(`{"type":"result","subtype":"success","result":"Hi","session_id":"s1","is_error":false,"num_turns":1,"total_cost_usd":0.01}`)

	mock := transport.NewMockTransport(msg, result)
	mock.Connect(nil)

	stream := agentsdk.NewStreamFromTransport(mock)
	defer stream.Close()

	res, err := stream.Result()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Result != "Hi" {
		t.Errorf("expected 'Hi', got %q", res.Result)
	}
	if res.TotalCostUSD == nil || *res.TotalCostUSD != 0.01 {
		t.Errorf("expected cost 0.01, got %v", res.TotalCostUSD)
	}
}

func TestStream_InvalidJSON(t *testing.T) {
	msg := json.RawMessage(`not json`)

	mock := transport.NewMockTransport(msg)
	mock.Connect(nil)

	stream := agentsdk.NewStreamFromTransport(mock)
	defer stream.Close()

	if stream.Next() {
		t.Fatal("expected Next() to return false on invalid JSON")
	}
	if stream.Err() == nil {
		t.Fatal("expected parse error")
	}
}

func TestStream_EmptyStream(t *testing.T) {
	mock := transport.NewMockTransport()
	mock.Connect(nil)

	stream := agentsdk.NewStreamFromTransport(mock)
	defer stream.Close()

	if stream.Next() {
		t.Fatal("expected Next() to return false on empty stream")
	}
	if stream.Err() != nil {
		t.Fatalf("unexpected error: %v", stream.Err())
	}
}

func TestStream_ToolUseBlock(t *testing.T) {
	msg := json.RawMessage(`{"type":"assistant","content":[{"type":"tool_use","id":"tu1","name":"Read","input":{"file_path":"/tmp/x"}}],"session_id":"s1"}`)

	mock := transport.NewMockTransport(msg)
	mock.Connect(nil)

	stream := agentsdk.NewStreamFromTransport(mock)
	defer stream.Close()

	if !stream.Next() {
		t.Fatal("expected message")
	}
	assistant, ok := stream.Current().AsAssistant()
	if !ok {
		t.Fatal("expected assistant message")
	}
	toolUse, ok := assistant.Content[0].AsToolUse()
	if !ok {
		t.Fatal("expected tool_use block")
	}
	if toolUse.Name != "Read" {
		t.Errorf("expected tool name 'Read', got %q", toolUse.Name)
	}
	if toolUse.Input["file_path"] != "/tmp/x" {
		t.Errorf("expected file_path '/tmp/x', got %v", toolUse.Input["file_path"])
	}
}

func TestStream_RateLimitEvent(t *testing.T) {
	msg := json.RawMessage(`{"type":"rate_limit","rate_limit_info":{"status":"allowed_warning","resets_at":"2026-03-31T12:00:00Z","rate_limit_type":"five_hour","utilization":0.85},"session_id":"s1","uuid":"u1"}`)

	mock := transport.NewMockTransport(msg)
	mock.Connect(nil)

	stream := agentsdk.NewStreamFromTransport(mock)
	defer stream.Close()

	if !stream.Next() {
		t.Fatal("expected message")
	}
	rl, ok := stream.Current().AsRateLimit()
	if !ok {
		t.Fatal("expected rate_limit message")
	}
	if rl.RateLimitInfo.Status != "allowed_warning" {
		t.Errorf("expected status 'allowed_warning', got %q", rl.RateLimitInfo.Status)
	}
	if rl.RateLimitInfo.Utilization != 0.85 {
		t.Errorf("expected utilization 0.85, got %v", rl.RateLimitInfo.Utilization)
	}
}

func TestBuildCLIArgs_BasicOptions(t *testing.T) {
	args := agentsdk.BuildCLIArgsForTest(agentsdk.TestQueryConfig{
		Model:          "opus",
		MaxTurns:       10,
		PermissionMode: agentsdk.PermissionBypassAll,
		AllowedTools:   []string{"Read", "Write"},
	})

	want := map[string]bool{
		"--model":           true,
		"opus":              true,
		"--max-turns":       true,
		"10":                true,
		"--permission-mode": true,
		"--allowedTools":    true,
		"Read":              true,
		"Write":             true,
	}
	for _, arg := range args {
		delete(want, arg)
	}
	// permission mode value
	delete(want, string(agentsdk.PermissionBypassAll))

	for missing := range want {
		t.Errorf("expected arg %q in CLI args", missing)
	}
}
