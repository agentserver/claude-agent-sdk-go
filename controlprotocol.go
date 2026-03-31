package agentsdk

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/anthropics/claude-agent-sdk-go/internal/transport"
)

// controlHandler processes control_request messages from the claude process.
// It routes hook callbacks, tool permission checks, and SDK MCP tool calls.
type controlHandler struct {
	config      *queryConfig
	tp          transport.Transport
	mu          sync.Mutex
	mcpServers  map[string]*McpSdkServer // SDK MCP servers keyed by server name
	callbackMap map[string]HookCallback  // callback_id → callback function (populated during initialize)
	reqCounter  atomic.Int64             // for generating request IDs
}

func newControlHandler(cfg *queryConfig, tp transport.Transport) *controlHandler {
	h := &controlHandler{
		config:      cfg,
		tp:          tp,
		mcpServers:  make(map[string]*McpSdkServer),
		callbackMap: make(map[string]HookCallback),
	}

	// Collect SDK MCP servers.
	for name, srv := range cfg.mcpServers {
		if srv.SDK != nil {
			h.mcpServers[name] = srv.SDK
		}
	}

	return h
}

// sendInitialize sends the initialize control request to register hooks and agents
// with the CLI process. This must be called after Connect() and before sending
// the first user message.
func (h *controlHandler) sendInitialize(ctx context.Context) error {
	request := map[string]any{
		"subtype": "initialize",
	}

	// Build hooks config with callback IDs.
	if len(h.config.hooks) > 0 {
		hooksConfig := map[string]any{}
		callbackIndex := 0
		for event, matchers := range h.config.hooks {
			var matcherConfigs []map[string]any
			for _, matcher := range matchers {
				var callbackIDs []string
				for _, hook := range matcher.Hooks {
					id := fmt.Sprintf("hook_%d", callbackIndex)
					h.callbackMap[id] = hook
					callbackIDs = append(callbackIDs, id)
					callbackIndex++
				}
				mc := map[string]any{
					"matcher":         matcher.Matcher,
					"hookCallbackIds": callbackIDs,
				}
				if matcher.Timeout > 0 {
					mc["timeout"] = matcher.Timeout.Milliseconds()
				}
				matcherConfigs = append(matcherConfigs, mc)
			}
			hooksConfig[string(event)] = matcherConfigs
		}
		request["hooks"] = hooksConfig
	}

	// Include agents.
	if len(h.config.agents) > 0 {
		request["agents"] = h.config.agents
	}

	// Send initialize request.
	reqID := fmt.Sprintf("sdk_init_%d", h.reqCounter.Add(1))
	msg := map[string]any{
		"type":       "control_request",
		"request_id": reqID,
		"request":    request,
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal initialize: %w", err)
	}
	return h.tp.Write(string(data))
}

// controlRequest is the envelope for a control request from the CLI.
type controlRequest struct {
	Type      string          `json:"type"`       // "control_request"
	RequestID string          `json:"request_id"`
	Request   json.RawMessage `json:"request"`
}

// controlRequestPayload extracts the subtype from the inner request.
type controlRequestPayload struct {
	Subtype string `json:"subtype"` // "hook_callback", "can_use_tool", "mcp_message", etc.
}

// handleMessage checks if a raw message is a control_request and handles it.
// Returns true if the message was handled (caller should not forward it).
func (h *controlHandler) handleMessage(ctx context.Context, raw json.RawMessage) bool {
	var req controlRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		return false
	}
	if req.Type != "control_request" {
		return false
	}

	var payload controlRequestPayload
	if err := json.Unmarshal(req.Request, &payload); err != nil {
		h.sendControlError(req.RequestID, err.Error())
		return true
	}

	switch payload.Subtype {
	case "hook_callback":
		go h.handleHookCallback(ctx, req.RequestID, req.Request)
	case "can_use_tool":
		go h.handleCanUseTool(ctx, req.RequestID, req.Request)
	case "mcp_message":
		go h.handleMcpMessage(ctx, req.RequestID, req.Request)
	default:
		// Unknown control request — respond with empty result to unblock.
		h.sendControlResponse(req.RequestID, map[string]any{})
	}

	return true
}

// handleHookCallback dispatches a hook callback by callback_id.
func (h *controlHandler) handleHookCallback(ctx context.Context, reqID string, payload json.RawMessage) {
	var req struct {
		Subtype    string    `json:"subtype"`
		CallbackID string    `json:"callback_id"`
		Input      HookInput `json:"input"`
		ToolUseID  string    `json:"tool_use_id,omitempty"`
	}
	if err := json.Unmarshal(payload, &req); err != nil {
		h.sendControlResponse(reqID, map[string]any{"continue": true})
		return
	}

	callback, ok := h.callbackMap[req.CallbackID]
	if !ok {
		h.sendControlResponse(reqID, map[string]any{"continue": true})
		return
	}

	output, err := callback(ctx, req.Input, req.ToolUseID)
	if err != nil {
		h.sendControlResponse(reqID, map[string]any{
			"decision": "block",
			"reason":   err.Error(),
		})
		return
	}

	resp, _ := json.Marshal(output)
	var respMap map[string]any
	json.Unmarshal(resp, &respMap)
	h.sendControlResponse(reqID, respMap)
}

// handleCanUseTool dispatches a tool permission check to the registered callback.
func (h *controlHandler) handleCanUseTool(ctx context.Context, reqID string, payload json.RawMessage) {
	if h.config.canUseTool == nil {
		h.sendControlResponse(reqID, map[string]any{"behavior": "allow"})
		return
	}

	var req struct {
		Subtype               string             `json:"subtype"`
		ToolName              string             `json:"tool_name"`
		Input                 map[string]any     `json:"input"`
		ToolUseID             string             `json:"tool_use_id"`
		AgentID               string             `json:"agent_id,omitempty"`
		PermissionSuggestions []PermissionUpdate `json:"permission_suggestions,omitempty"`
	}
	if err := json.Unmarshal(payload, &req); err != nil {
		h.sendControlResponse(reqID, map[string]any{"behavior": "allow"})
		return
	}

	permCtx := ToolPermissionContext{
		ToolUseID:   req.ToolUseID,
		AgentID:     req.AgentID,
		Suggestions: req.PermissionSuggestions,
	}

	result, err := h.config.canUseTool(req.ToolName, req.Input, permCtx)
	if err != nil {
		h.sendControlResponse(reqID, map[string]any{
			"behavior": "deny",
			"message":  err.Error(),
		})
		return
	}

	resp, _ := json.Marshal(result)
	var respMap map[string]any
	json.Unmarshal(resp, &respMap)
	h.sendControlResponse(reqID, respMap)
}

// handleMcpMessage routes JSONRPC messages to in-process SDK MCP servers.
func (h *controlHandler) handleMcpMessage(ctx context.Context, reqID string, payload json.RawMessage) {
	var req struct {
		Subtype    string          `json:"subtype"`
		ServerName string          `json:"server_name"`
		Message    json.RawMessage `json:"message"`
	}
	if err := json.Unmarshal(payload, &req); err != nil {
		h.sendControlError(reqID, err.Error())
		return
	}

	server, ok := h.mcpServers[req.ServerName]
	if !ok {
		h.sendControlError(reqID, fmt.Sprintf("unknown MCP server: %s", req.ServerName))
		return
	}

	// Parse JSONRPC message.
	var rpc struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      any             `json:"id"`
		Method  string          `json:"method"`
		Params  json.RawMessage `json:"params"`
	}
	if err := json.Unmarshal(req.Message, &rpc); err != nil {
		h.sendControlError(reqID, err.Error())
		return
	}

	var rpcResponse map[string]any

	switch rpc.Method {
	case "initialize":
		rpcResponse = map[string]any{
			"jsonrpc": "2.0",
			"id":      rpc.ID,
			"result": map[string]any{
				"protocolVersion": "2024-11-05",
				"capabilities":   map[string]any{"tools": map[string]any{}},
				"serverInfo": map[string]any{
					"name":    server.Name,
					"version": server.Version,
				},
			},
		}

	case "notifications/initialized":
		rpcResponse = map[string]any{
			"jsonrpc": "2.0",
			"result":  map[string]any{},
		}

	case "tools/list":
		var tools []map[string]any
		for _, t := range server.Tools {
			toolDef := map[string]any{
				"name":        t.Name,
				"description": t.Description,
				"inputSchema": json.RawMessage(t.InputSchema),
			}
			if t.Annotations != nil {
				toolDef["annotations"] = t.Annotations
			}
			tools = append(tools, toolDef)
		}
		rpcResponse = map[string]any{
			"jsonrpc": "2.0",
			"id":      rpc.ID,
			"result":  map[string]any{"tools": tools},
		}

	case "tools/call":
		var params struct {
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		}
		if err := json.Unmarshal(rpc.Params, &params); err != nil {
			rpcResponse = map[string]any{
				"jsonrpc": "2.0",
				"id":      rpc.ID,
				"error":   map[string]any{"code": -32603, "message": err.Error()},
			}
			break
		}

		// Find the tool.
		var tool *McpTool
		for i := range server.Tools {
			if server.Tools[i].Name == params.Name {
				tool = &server.Tools[i]
				break
			}
		}
		if tool == nil {
			rpcResponse = map[string]any{
				"jsonrpc": "2.0",
				"id":      rpc.ID,
				"error":   map[string]any{"code": -32601, "message": fmt.Sprintf("unknown tool: %s", params.Name)},
			}
			break
		}

		result, err := tool.Handler(ctx, params.Arguments)
		if err != nil {
			rpcResponse = map[string]any{
				"jsonrpc": "2.0",
				"id":      rpc.ID,
				"result": map[string]any{
					"content": []map[string]any{{"type": "text", "text": err.Error()}},
					"isError": true,
				},
			}
			break
		}

		resultMap, _ := json.Marshal(result)
		var resultAny map[string]any
		json.Unmarshal(resultMap, &resultAny)
		rpcResponse = map[string]any{
			"jsonrpc": "2.0",
			"id":      rpc.ID,
			"result":  resultAny,
		}

	default:
		rpcResponse = map[string]any{
			"jsonrpc": "2.0",
			"id":      rpc.ID,
			"error":   map[string]any{"code": -32601, "message": fmt.Sprintf("method not found: %s", rpc.Method)},
		}
	}

	h.sendControlResponse(reqID, map[string]any{"mcp_response": rpcResponse})
}

// sendControlResponse writes a success control_response message to the transport.
func (h *controlHandler) sendControlResponse(reqID string, result map[string]any) {
	msg := map[string]any{
		"type": "control_response",
		"response": map[string]any{
			"subtype":    "success",
			"request_id": reqID,
			"response":   result,
		},
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.tp.Write(string(data))
}

// sendControlError writes an error control_response message to the transport.
func (h *controlHandler) sendControlError(reqID string, errMsg string) {
	msg := map[string]any{
		"type": "control_response",
		"response": map[string]any{
			"subtype":    "error",
			"request_id": reqID,
			"error":      errMsg,
		},
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.tp.Write(string(data))
}
