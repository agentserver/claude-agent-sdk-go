package transport

import (
	"context"
	"encoding/json"
)

// Transport abstracts the communication channel with the claude process.
// The default implementation spawns a subprocess, but custom implementations
// can connect to remote claude instances.
type Transport interface {
	// Connect starts the transport and begins reading messages.
	Connect(ctx context.Context) error

	// Write sends a string (NDJSON line) to the claude process stdin.
	Write(data string) error

	// ReadMessages returns a channel that yields raw JSON messages from stdout.
	// The channel is closed when the process exits or the transport is closed.
	ReadMessages() <-chan json.RawMessage

	// Close shuts down the transport and releases resources.
	// It performs graceful shutdown: close stdin → wait → SIGTERM → wait → SIGKILL.
	Close() error

	// EndInput closes the stdin pipe, signaling EOF to the claude process.
	EndInput() error

	// IsReady returns true if the transport is connected and ready.
	IsReady() bool

	// ExitError returns the process exit error, or nil if still running or exited cleanly.
	ExitError() error
}
