package agentsdk

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
)

// McpServerConfig is a union of MCP server configuration types.
// Exactly one field should be set.
type McpServerConfig struct {
	// Stdio launches an MCP server as a subprocess.
	Stdio *McpStdioServerConfig `json:"stdio,omitempty"`
	// SSE connects to an MCP server over Server-Sent Events.
	SSE *McpSSEServerConfig `json:"sse,omitempty"`
	// HTTP connects to an MCP server over HTTP.
	HTTP *McpHTTPServerConfig `json:"http,omitempty"`
	// Proxy connects through the claude.ai proxy (for managed MCP servers).
	Proxy *McpProxyServerConfig `json:"proxy,omitempty"`
	// SDK is an in-process MCP server defined in Go.
	SDK *McpSdkServer `json:"-"` // Not serialized — handled by control protocol
}

// MarshalJSON implements custom marshalling for the union.
func (c McpServerConfig) MarshalJSON() ([]byte, error) {
	if c.Stdio != nil {
		return json.Marshal(struct {
			Type string `json:"type"`
			*McpStdioServerConfig
		}{"stdio", c.Stdio})
	}
	if c.SSE != nil {
		return json.Marshal(struct {
			Type string `json:"type"`
			*McpSSEServerConfig
		}{"sse", c.SSE})
	}
	if c.HTTP != nil {
		return json.Marshal(struct {
			Type string `json:"type"`
			*McpHTTPServerConfig
		}{"http", c.HTTP})
	}
	if c.Proxy != nil {
		return json.Marshal(struct {
			Type string `json:"type"`
			*McpProxyServerConfig
		}{"claudeai-proxy", c.Proxy})
	}
	return []byte("{}"), nil
}

// McpStdioServerConfig launches an MCP server as a child process.
type McpStdioServerConfig struct {
	Command string            `json:"command"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}

// McpSSEServerConfig connects to an MCP server via Server-Sent Events.
type McpSSEServerConfig struct {
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
}

// McpHTTPServerConfig connects to an MCP server via HTTP.
type McpHTTPServerConfig struct {
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
}

// McpProxyServerConfig connects through the claude.ai proxy for managed MCP servers.
type McpProxyServerConfig struct {
	URL      string `json:"url"`                // Proxy URL
	ServerID string `json:"id,omitempty"`        // Managed server identifier
}

// McpSdkServer is an in-process MCP server defined in Go code.
// Tools are registered via the Tool() helper function.
type McpSdkServer struct {
	Name    string
	Version string
	Tools   []McpTool
}

// McpTool defines a single tool exposed by an SDK MCP server.
type McpTool struct {
	Name        string
	Description string
	InputSchema json.RawMessage // JSON Schema for the tool input
	Handler     func(ctx context.Context, input json.RawMessage) (*McpToolResult, error)
	Annotations *ToolAnnotations // Optional MCP tool annotations
}

// ToolAnnotations provides metadata hints about a tool's behavior.
type ToolAnnotations struct {
	ReadOnly    bool `json:"readOnly,omitempty"`    // Tool only reads data, no side effects
	Destructive bool `json:"destructive,omitempty"` // Tool may perform destructive operations
	OpenWorld   bool `json:"openWorld,omitempty"`   // Tool interacts with external systems
}

// McpToolResult is the response from an MCP tool handler.
type McpToolResult struct {
	Content []McpToolContent `json:"content"`
	IsError bool             `json:"isError,omitempty"`
}

// McpToolContent is a content item in an MCP tool result.
type McpToolContent struct {
	Type string `json:"type"` // "text" or "image"
	Text string `json:"text,omitempty"`
}

// McpServerStatus reports the connection status of an MCP server.
type McpServerStatus struct {
	Name       string         `json:"name"`
	Status     string         `json:"status"` // "connected", "failed", "pending", "needs-auth", "needs-approval", "disabled"
	ServerInfo *McpServerInfo `json:"serverInfo,omitempty"`
	Error      string         `json:"error,omitempty"`
	Config     map[string]any `json:"config,omitempty"` // Server configuration snapshot
	Scope      string         `json:"scope,omitempty"`  // "user", "project", "local"
	Tools      []McpToolInfo  `json:"tools,omitempty"`
}

// McpServerInfo identifies a connected MCP server.
type McpServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// McpToolInfo describes a tool exposed by an MCP server.
type McpToolInfo struct {
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	Annotations *ToolAnnotations `json:"annotations,omitempty"`
}

// ToolOption configures an McpTool created by the Tool() helper.
type ToolOption func(*McpTool)

// WithAnnotations sets MCP tool annotations on a tool.
func WithAnnotations(annotations ToolAnnotations) ToolOption {
	return func(t *McpTool) { t.Annotations = &annotations }
}

// Tool creates a typed McpTool with automatic JSON schema generation from the
// type parameter T. The handler receives a decoded T value.
func Tool[T any](name, description string, handler func(context.Context, T) (*McpToolResult, error), opts ...ToolOption) McpTool {
	// Generate JSON schema from T by marshalling a zero value.
	var zero T
	schema, _ := json.Marshal(schemaFromType(zero))

	tool := McpTool{
		Name:        name,
		Description: description,
		InputSchema: schema,
		Handler: func(ctx context.Context, input json.RawMessage) (*McpToolResult, error) {
			var v T
			if err := json.Unmarshal(input, &v); err != nil {
				return nil, err
			}
			return handler(ctx, v)
		},
	}
	for _, opt := range opts {
		opt(&tool)
	}
	return tool
}

// CreateSdkMcpServer is a convenience constructor for McpSdkServer.
func CreateSdkMcpServer(name, version string, tools ...McpTool) *McpSdkServer {
	return &McpSdkServer{
		Name:    name,
		Version: version,
		Tools:   tools,
	}
}

// schemaFromType generates a JSON schema object from a Go struct using reflection.
// Supports struct fields with json tags. Nested structs are expanded recursively.
func schemaFromType(v any) map[string]any {
	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return map[string]any{"type": "object"}
	}

	properties := map[string]any{}
	var required []string

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}

		name := field.Name
		if tag := field.Tag.Get("json"); tag != "" {
			parts := strings.Split(tag, ",")
			if parts[0] == "-" {
				continue
			}
			if parts[0] != "" {
				name = parts[0]
			}
			// Fields without omitempty are required.
			hasOmitempty := false
			for _, p := range parts[1:] {
				if p == "omitempty" {
					hasOmitempty = true
				}
			}
			if !hasOmitempty {
				required = append(required, name)
			}
		} else {
			required = append(required, name)
		}

		properties[name] = goTypeToJSONSchema(field.Type)
	}

	schema := map[string]any{
		"type":       "object",
		"properties": properties,
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

// goTypeToJSONSchema maps Go types to JSON Schema type strings.
func goTypeToJSONSchema(t reflect.Type) map[string]any {
	// Dereference pointers.
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	switch t.Kind() {
	case reflect.String:
		return map[string]any{"type": "string"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return map[string]any{"type": "integer"}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return map[string]any{"type": "integer"}
	case reflect.Float32, reflect.Float64:
		return map[string]any{"type": "number"}
	case reflect.Bool:
		return map[string]any{"type": "boolean"}
	case reflect.Slice:
		return map[string]any{"type": "array", "items": goTypeToJSONSchema(t.Elem())}
	case reflect.Map:
		if t.Key().Kind() == reflect.String {
			return map[string]any{"type": "object", "additionalProperties": goTypeToJSONSchema(t.Elem())}
		}
		return map[string]any{"type": "object"}
	case reflect.Struct:
		return schemaFromType(reflect.New(t).Elem().Interface())
	case reflect.Interface:
		return map[string]any{} // any type — no constraint
	default:
		return map[string]any{"type": "object"}
	}
}
