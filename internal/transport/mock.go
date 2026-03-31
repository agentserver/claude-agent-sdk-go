package transport

import (
	"context"
	"encoding/json"
	"sync"
)

// MockTransport is a test double for Transport.
type MockTransport struct {
	messages []json.RawMessage
	msgCh    chan json.RawMessage
	written  []string
	ready    bool
	mu       sync.Mutex
}

// NewMockTransport creates a MockTransport preloaded with messages.
func NewMockTransport(messages ...json.RawMessage) *MockTransport {
	return &MockTransport{
		messages: messages,
		msgCh:    make(chan json.RawMessage, len(messages)),
	}
}

func (m *MockTransport) Connect(_ context.Context) error {
	m.mu.Lock()
	m.ready = true
	m.mu.Unlock()
	for _, msg := range m.messages {
		m.msgCh <- msg
	}
	close(m.msgCh)
	return nil
}

func (m *MockTransport) Write(data string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.written = append(m.written, data)
	return nil
}

func (m *MockTransport) ReadMessages() <-chan json.RawMessage {
	return m.msgCh
}

func (m *MockTransport) Close() error {
	m.mu.Lock()
	m.ready = false
	m.mu.Unlock()
	return nil
}

func (m *MockTransport) EndInput() error { return nil }

func (m *MockTransport) ExitError() error { return nil }

func (m *MockTransport) IsReady() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.ready
}

// Written returns all strings written to the transport.
func (m *MockTransport) Written() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]string{}, m.written...)
}
