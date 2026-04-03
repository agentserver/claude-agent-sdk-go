package agentsdk_test

import (
	"encoding/json"
	"testing"

	agentsdk "github.com/anthropics/claude-agent-sdk-go"
	"github.com/anthropics/claude-agent-sdk-go/internal/transport"
)

func TestStream_SingleAssistantMessage(t *testing.T) {
	// Wire format: content/model nested under "message" field.
	msg := json.RawMessage(`{"type":"assistant","message":{"id":"msg1","type":"message","role":"assistant","content":[{"type":"text","text":"Hello!"}],"model":"claude-sonnet-4-6","stop_reason":"end_turn"},"parent_tool_use_id":null,"session_id":"s1","uuid":"u1"}`)
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
	if len(assistant.Message.Content) == 0 {
		t.Fatal("expected content blocks")
	}
	text, ok := assistant.Message.Content[0].AsText()
	if !ok {
		t.Fatal("expected text block")
	}
	if text.Text != "Hello!" {
		t.Errorf("expected 'Hello!', got %q", text.Text)
	}
	if assistant.Message.Model != "claude-sonnet-4-6" {
		t.Errorf("expected model 'claude-sonnet-4-6', got %q", assistant.Message.Model)
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
	msg := json.RawMessage(`{"type":"assistant","message":{"content":[{"type":"text","text":"Hi"}],"model":"claude-sonnet-4-6"},"parent_tool_use_id":null,"session_id":"s1","uuid":"u1"}`)
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
	if res.TotalCostUSD != 0.01 {
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
	msg := json.RawMessage(`{"type":"assistant","message":{"content":[{"type":"tool_use","id":"tu1","name":"Read","input":{"file_path":"/tmp/x"}}],"model":"claude-sonnet-4-6"},"parent_tool_use_id":null,"session_id":"s1","uuid":"u1"}`)

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
	toolUse, ok := assistant.Message.Content[0].AsToolUse()
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

// FIX #2: RateLimitEvent uses "rate_limit_event" (not "rate_limit")
func TestStream_RateLimitEvent(t *testing.T) {
	msg := json.RawMessage(`{"type":"rate_limit_event","rate_limit_info":{"status":"allowed_warning","resetsAt":1711886400000,"rateLimitType":"five_hour","utilization":0.85},"session_id":"s1","uuid":"u1"}`)

	mock := transport.NewMockTransport(msg)
	mock.Connect(nil)

	stream := agentsdk.NewStreamFromTransport(mock)
	defer stream.Close()

	if !stream.Next() {
		t.Fatal("expected message")
	}
	rl, ok := stream.Current().AsRateLimit()
	if !ok {
		t.Fatal("expected rate_limit_event message")
	}
	if rl.RateLimitInfo.Status != "allowed_warning" {
		t.Errorf("expected status 'allowed_warning', got %q", rl.RateLimitInfo.Status)
	}
	if rl.RateLimitInfo.Utilization != 0.85 {
		t.Errorf("expected utilization 0.85, got %v", rl.RateLimitInfo.Utilization)
	}
	if rl.RateLimitInfo.RateLimitType != "five_hour" {
		t.Errorf("expected rateLimitType 'five_hour', got %q", rl.RateLimitInfo.RateLimitType)
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

// --- System subtype tests ---

func TestStream_SystemSubtypes(t *testing.T) {
	tests := []struct {
		name  string
		json  string
		check func(t *testing.T, msg agentsdk.SDKMessage)
	}{
		{
			name: "api_retry",
			json: `{"type":"system","subtype":"api_retry","attempt":1,"max_retries":3,"retry_delay_ms":5000,"error_status":429,"error":"rate_limit","uuid":"u1","session_id":"s1"}`,
			check: func(t *testing.T, msg agentsdk.SDKMessage) {
				retry, ok := msg.AsAPIRetry()
				if !ok {
					t.Fatal("AsAPIRetry returned false")
				}
				if retry.Attempt != 1 || retry.MaxRetries != 3 {
					t.Errorf("expected attempt=1 max_retries=3, got %d/%d", retry.Attempt, retry.MaxRetries)
				}
				if retry.ErrorStatus == nil || *retry.ErrorStatus != 429 {
					t.Errorf("expected error_status 429, got %v", retry.ErrorStatus)
				}
			},
		},
		{
			name: "compact_boundary",
			json: `{"type":"system","subtype":"compact_boundary","compact_metadata":{"trigger":"auto","pre_tokens":50000},"uuid":"u1","session_id":"s1"}`,
			check: func(t *testing.T, msg agentsdk.SDKMessage) {
				cb, ok := msg.AsCompactBoundary()
				if !ok {
					t.Fatal("AsCompactBoundary returned false")
				}
				if cb.CompactMetadata.Trigger != "auto" {
					t.Errorf("expected trigger 'auto', got %q", cb.CompactMetadata.Trigger)
				}
			},
		},
		{
			name: "hook_started",
			json: `{"type":"system","subtype":"hook_started","hook_id":"h1","hook_name":"my-hook","hook_event":"PreToolUse","uuid":"u1","session_id":"s1"}`,
			check: func(t *testing.T, msg agentsdk.SDKMessage) {
				hs, ok := msg.AsHookStarted()
				if !ok {
					t.Fatal("AsHookStarted returned false")
				}
				if hs.HookName != "my-hook" {
					t.Errorf("expected hook_name 'my-hook', got %q", hs.HookName)
				}
			},
		},
		{
			name: "hook_response",
			json: `{"type":"system","subtype":"hook_response","hook_id":"h1","hook_name":"my-hook","hook_event":"PostToolUse","output":"ok","stdout":"","stderr":"","exit_code":0,"outcome":"success","uuid":"u1","session_id":"s1"}`,
			check: func(t *testing.T, msg agentsdk.SDKMessage) {
				hr, ok := msg.AsHookResponse()
				if !ok {
					t.Fatal("AsHookResponse returned false")
				}
				if hr.Outcome != "success" {
					t.Errorf("expected outcome 'success', got %q", hr.Outcome)
				}
			},
		},
		{
			name: "elicitation_complete",
			json: `{"type":"system","subtype":"elicitation_complete","mcp_server_name":"auth-server","elicitation_id":"e1","uuid":"u1","session_id":"s1"}`,
			check: func(t *testing.T, msg agentsdk.SDKMessage) {
				ec, ok := msg.AsElicitationComplete()
				if !ok {
					t.Fatal("AsElicitationComplete returned false")
				}
				if ec.McpServerName != "auth-server" {
					t.Errorf("expected mcp_server_name 'auth-server', got %q", ec.McpServerName)
				}
			},
		},
		{
			// FIX #4: task_started is type="system", subtype="task_started"
			name: "task_started",
			json: `{"type":"system","subtype":"task_started","task_id":"t1","description":"Running tests","task_type":"agent","tool_use_id":"tu1","uuid":"u1","session_id":"s1"}`,
			check: func(t *testing.T, msg agentsdk.SDKMessage) {
				ts, ok := msg.AsTaskStarted()
				if !ok {
					t.Fatal("AsTaskStarted returned false")
				}
				if ts.TaskID != "t1" {
					t.Errorf("expected task_id 't1', got %q", ts.TaskID)
				}
				if ts.Description != "Running tests" {
					t.Errorf("expected description 'Running tests', got %q", ts.Description)
				}
			},
		},
		{
			// FIX #4: task_notification is type="system", subtype="task_notification"
			name: "task_notification",
			json: `{"type":"system","subtype":"task_notification","task_id":"t1","status":"completed","output_file":"/tmp/out","summary":"Done","uuid":"u1","session_id":"s1"}`,
			check: func(t *testing.T, msg agentsdk.SDKMessage) {
				tn, ok := msg.AsTaskNotification()
				if !ok {
					t.Fatal("AsTaskNotification returned false")
				}
				if tn.Status != "completed" {
					t.Errorf("expected status 'completed', got %q", tn.Status)
				}
			},
		},
		{
			// FIX #7: status message has status field (not message/data)
			name: "status",
			json: `{"type":"system","subtype":"status","status":"compacting","uuid":"u1","session_id":"s1"}`,
			check: func(t *testing.T, msg agentsdk.SDKMessage) {
				st, ok := msg.AsStatus()
				if !ok {
					t.Fatal("AsStatus returned false")
				}
				if st.Status == nil || *st.Status != "compacting" {
					t.Errorf("expected status 'compacting', got %v", st.Status)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := transport.NewMockTransport(json.RawMessage(tt.json))
			mock.Connect(nil)

			stream := agentsdk.NewStreamFromTransport(mock)
			defer stream.Close()

			if !stream.Next() {
				t.Fatal("expected message")
			}
			// System subtype messages should still be accessible via AsSystem()
			if _, ok := stream.Current().AsSystem(); !ok {
				t.Error("expected AsSystem() to return true for system subtype")
			}
			tt.check(t, stream.Current())
		})
	}
}

func TestStream_NewTopLevelTypes(t *testing.T) {
	tests := []struct {
		name  string
		json  string
		check func(t *testing.T, msg agentsdk.SDKMessage)
	}{
		{
			name: "auth_status",
			json: `{"type":"auth_status","isAuthenticating":true,"output":["Authenticating..."],"uuid":"u1","session_id":"s1"}`,
			check: func(t *testing.T, msg agentsdk.SDKMessage) {
				as, ok := msg.AsAuthStatus()
				if !ok {
					t.Fatal("AsAuthStatus returned false")
				}
				if !as.IsAuthenticating {
					t.Error("expected isAuthenticating true")
				}
			},
		},
		{
			name: "prompt_suggestion",
			json: `{"type":"prompt_suggestion","suggestion":"Try asking about X","uuid":"u1","session_id":"s1"}`,
			check: func(t *testing.T, msg agentsdk.SDKMessage) {
				ps, ok := msg.AsPromptSuggestion()
				if !ok {
					t.Fatal("AsPromptSuggestion returned false")
				}
				if ps.Suggestion != "Try asking about X" {
					t.Errorf("expected suggestion 'Try asking about X', got %q", ps.Suggestion)
				}
			},
		},
		{
			// FIX #1: UserMessageReplay has type="user" with isReplay=true
			name: "user_replay",
			json: `{"type":"user","message":{"role":"user","content":"Hello"},"parent_tool_use_id":null,"isReplay":true,"uuid":"u1","session_id":"s1"}`,
			check: func(t *testing.T, msg agentsdk.SDKMessage) {
				rp, ok := msg.AsUserReplay()
				if !ok {
					t.Fatal("AsUserReplay returned false")
				}
				if !rp.IsReplay {
					t.Error("expected isReplay true")
				}
				if rp.Message.Role != "user" {
					t.Errorf("expected role 'user', got %q", rp.Message.Role)
				}
			},
		},
		{
			// Verify regular user message does NOT match AsUserReplay
			name: "user_not_replay",
			json: `{"type":"user","message":{"role":"user","content":"Hello"},"parent_tool_use_id":null,"uuid":"u1","session_id":"s1"}`,
			check: func(t *testing.T, msg agentsdk.SDKMessage) {
				_, ok := msg.AsUserReplay()
				if ok {
					t.Fatal("AsUserReplay should return false for non-replay user message")
				}
				u, ok := msg.AsUser()
				if !ok {
					t.Fatal("AsUser should return true")
				}
				if u.IsReplay {
					t.Error("expected isReplay false for normal user message")
				}
			},
		},
		{
			// FIX #5: ToolProgressMessage with correct fields
			name: "tool_progress",
			json: `{"type":"tool_progress","tool_use_id":"tu1","tool_name":"Bash","parent_tool_use_id":null,"elapsed_time_seconds":5.2,"uuid":"u1","session_id":"s1"}`,
			check: func(t *testing.T, msg agentsdk.SDKMessage) {
				tp, ok := msg.AsToolProgress()
				if !ok {
					t.Fatal("AsToolProgress returned false")
				}
				if tp.ToolName != "Bash" {
					t.Errorf("expected tool_name 'Bash', got %q", tp.ToolName)
				}
				if tp.ElapsedTimeSeconds != 5.2 {
					t.Errorf("expected elapsed_time_seconds 5.2, got %v", tp.ElapsedTimeSeconds)
				}
			},
		},
		{
			// FIX #6: ToolUseSummaryMessage with correct fields
			name: "tool_use_summary",
			json: `{"type":"tool_use_summary","summary":"Read 3 files","preceding_tool_use_ids":["tu1","tu2","tu3"],"uuid":"u1","session_id":"s1"}`,
			check: func(t *testing.T, msg agentsdk.SDKMessage) {
				ts, ok := msg.AsToolUseSummary()
				if !ok {
					t.Fatal("AsToolUseSummary returned false")
				}
				if ts.Summary != "Read 3 files" {
					t.Errorf("expected summary 'Read 3 files', got %q", ts.Summary)
				}
				if len(ts.PrecedingToolUseIDs) != 3 {
					t.Errorf("expected 3 preceding_tool_use_ids, got %d", len(ts.PrecedingToolUseIDs))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := transport.NewMockTransport(json.RawMessage(tt.json))
			mock.Connect(nil)

			stream := agentsdk.NewStreamFromTransport(mock)
			defer stream.Close()

			if !stream.Next() {
				t.Fatal("expected message")
			}
			tt.check(t, stream.Current())
		})
	}
}

func TestSDKMessage_SubtypeExtracted(t *testing.T) {
	raw := json.RawMessage(`{"type":"system","subtype":"api_retry","attempt":1,"uuid":"u1","session_id":"s1"}`)
	var msg agentsdk.SDKMessage
	if err := json.Unmarshal(raw, &msg); err != nil {
		t.Fatal(err)
	}
	if msg.Type != "system" {
		t.Errorf("expected type 'system', got %q", msg.Type)
	}
	if msg.Subtype != "api_retry" {
		t.Errorf("expected subtype 'api_retry', got %q", msg.Subtype)
	}
}
