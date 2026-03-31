package agentsdk

import "encoding/json"

// ContentBlock is a tagged union of content block types within a message.
// Use the As* methods to access the concrete type.
type ContentBlock struct {
	Type string          `json:"type"` // "text", "thinking", "tool_use", "tool_result"
	Raw  json.RawMessage `json:"-"`    // Original JSON for advanced use cases
}

// UnmarshalJSON implements custom unmarshalling to capture raw JSON.
func (b *ContentBlock) UnmarshalJSON(data []byte) error {
	// Capture raw data.
	b.Raw = append(b.Raw[:0], data...)

	// Extract type field.
	var envelope struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return err
	}
	b.Type = envelope.Type
	return nil
}

// AsText returns the block as a TextBlock if Type == "text".
func (b ContentBlock) AsText() (*TextBlock, bool) {
	if b.Type != "text" {
		return nil, false
	}
	var t TextBlock
	if err := json.Unmarshal(b.Raw, &t); err != nil {
		return nil, false
	}
	return &t, true
}

// AsThinking returns the block as a ThinkingBlock if Type == "thinking".
func (b ContentBlock) AsThinking() (*ThinkingBlock, bool) {
	if b.Type != "thinking" {
		return nil, false
	}
	var t ThinkingBlock
	if err := json.Unmarshal(b.Raw, &t); err != nil {
		return nil, false
	}
	return &t, true
}

// AsToolUse returns the block as a ToolUseBlock if Type == "tool_use".
func (b ContentBlock) AsToolUse() (*ToolUseBlock, bool) {
	if b.Type != "tool_use" {
		return nil, false
	}
	var t ToolUseBlock
	if err := json.Unmarshal(b.Raw, &t); err != nil {
		return nil, false
	}
	return &t, true
}

// AsToolResult returns the block as a ToolResultBlock if Type == "tool_result".
func (b ContentBlock) AsToolResult() (*ToolResultBlock, bool) {
	if b.Type != "tool_result" {
		return nil, false
	}
	var t ToolResultBlock
	if err := json.Unmarshal(b.Raw, &t); err != nil {
		return nil, false
	}
	return &t, true
}

// TextBlock contains plain text content.
type TextBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// ThinkingBlock contains extended thinking content.
type ThinkingBlock struct {
	Type      string `json:"type"`
	Thinking  string `json:"thinking"`
	Signature string `json:"signature"`
}

// ToolUseBlock represents a tool invocation by the model.
type ToolUseBlock struct {
	Type  string         `json:"type"`
	ID    string         `json:"id"`
	Name  string         `json:"name"`
	Input map[string]any `json:"input"`
}

// ToolResultBlock contains the result of a tool invocation.
type ToolResultBlock struct {
	Type      string `json:"type"`
	ToolUseID string `json:"tool_use_id"`
	Content   any    `json:"content"` // string or []ContentBlock
	IsError   *bool  `json:"is_error,omitempty"`
}
