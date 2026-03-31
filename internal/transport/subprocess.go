package transport

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"
)

// SubprocessConfig holds configuration for spawning the claude process.
type SubprocessConfig struct {
	CLIPath       string
	Args          []string          // Additional CLI arguments
	Cwd           string            // Working directory
	Env           map[string]string // Additional environment variables
	MaxBufferSize int               // Max line size for stdout scanner (bytes)
	StderrFunc    func(string)      // Callback for stderr lines
}

// SubprocessTransport spawns a claude CLI subprocess and communicates
// via stdin (write) and stdout (read) using NDJSON.
type SubprocessTransport struct {
	config  SubprocessConfig
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	stdout  io.ReadCloser
	stderr  io.ReadCloser
	msgCh   chan json.RawMessage
	ready   bool
	mu      sync.Mutex
	cancel  context.CancelFunc
	done    chan struct{}
	exitErr error // Process exit error (non-zero exit code)
}

// NewSubprocessTransport creates a new SubprocessTransport.
func NewSubprocessTransport(config SubprocessConfig) *SubprocessTransport {
	if config.MaxBufferSize <= 0 {
		config.MaxBufferSize = 1 << 20 // 1MB default
	}
	return &SubprocessTransport{
		config: config,
		msgCh:  make(chan json.RawMessage, 64),
		done:   make(chan struct{}),
	}
}

// Connect spawns the claude subprocess and begins reading messages.
func (t *SubprocessTransport) Connect(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	t.cancel = cancel

	// Build command arguments.
	args := []string{
		"--print",
		"--output-format", "stream-json",
		"--input-format", "stream-json",
		"--verbose",
	}
	args = append(args, t.config.Args...)

	t.cmd = exec.CommandContext(ctx, t.config.CLIPath, args...)

	// Set working directory.
	if t.config.Cwd != "" {
		t.cmd.Dir = t.config.Cwd
	}

	// Set environment: inherit current env + additions.
	t.cmd.Env = os.Environ()
	for k, v := range t.config.Env {
		t.cmd.Env = append(t.cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	// Set up pipes.
	var err error
	t.stdin, err = t.cmd.StdinPipe()
	if err != nil {
		cancel()
		return fmt.Errorf("stdin pipe: %w", err)
	}
	t.stdout, err = t.cmd.StdoutPipe()
	if err != nil {
		cancel()
		return fmt.Errorf("stdout pipe: %w", err)
	}
	t.stderr, err = t.cmd.StderrPipe()
	if err != nil {
		cancel()
		return fmt.Errorf("stderr pipe: %w", err)
	}

	// Start the process.
	if err := t.cmd.Start(); err != nil {
		cancel()
		return fmt.Errorf("start claude: %w", err)
	}

	t.mu.Lock()
	t.ready = true
	t.mu.Unlock()

	// Read stdout in background.
	go t.readStdout()

	// Read stderr in background.
	go t.readStderr()

	// Wait for process exit in background.
	go func() {
		err := t.cmd.Wait()
		t.mu.Lock()
		t.ready = false
		t.exitErr = err
		t.mu.Unlock()
		close(t.done)
	}()

	return nil
}

// readStdout reads NDJSON lines from stdout and sends them to msgCh.
func (t *SubprocessTransport) readStdout() {
	scanner := bufio.NewScanner(t.stdout)
	scanner.Buffer(make([]byte, 0, t.config.MaxBufferSize), t.config.MaxBufferSize)

	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Validate it's JSON before sending.
		if !json.Valid([]byte(line)) {
			continue
		}
		t.msgCh <- json.RawMessage(line)
	}

	close(t.msgCh)
}

// readStderr reads stderr lines and calls the stderr callback.
func (t *SubprocessTransport) readStderr() {
	scanner := bufio.NewScanner(t.stderr)
	for scanner.Scan() {
		line := scanner.Text()
		if t.config.StderrFunc != nil {
			t.config.StderrFunc(line)
		}
	}
}

// Write sends a line to the claude process stdin, terminated with newline.
func (t *SubprocessTransport) Write(data string) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.ready || t.stdin == nil {
		return fmt.Errorf("transport not connected")
	}
	if !strings.HasSuffix(data, "\n") {
		data += "\n"
	}
	_, err := io.WriteString(t.stdin, data)
	return err
}

// ReadMessages returns the channel of parsed JSON messages from stdout.
func (t *SubprocessTransport) ReadMessages() <-chan json.RawMessage {
	return t.msgCh
}

// EndInput closes the stdin pipe, signaling EOF to the claude process.
func (t *SubprocessTransport) EndInput() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.stdin != nil {
		return t.stdin.Close()
	}
	return nil
}

// IsReady returns true if the transport is connected and the process is running.
func (t *SubprocessTransport) IsReady() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.ready
}

// ExitError returns the process exit error, or nil if still running or exited cleanly.
func (t *SubprocessTransport) ExitError() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.exitErr
}

// Close performs graceful shutdown of the claude process.
// Sequence: close stdin → wait 5s → SIGTERM → wait 2s → SIGKILL.
func (t *SubprocessTransport) Close() error {
	// Close stdin to signal EOF.
	t.EndInput()

	// Wait up to 5 seconds for natural exit.
	select {
	case <-t.done:
		return nil
	case <-time.After(5 * time.Second):
	}

	// Send SIGTERM (Unix) or kill immediately (Windows).
	if t.cmd.Process != nil {
		if runtime.GOOS == "windows" {
			t.cmd.Process.Kill()
			<-t.done
			if t.cancel != nil {
				t.cancel()
			}
			return nil
		}
		t.cmd.Process.Signal(syscall.SIGTERM)
	}

	// Wait up to 2 seconds.
	select {
	case <-t.done:
		return nil
	case <-time.After(2 * time.Second):
	}

	// Force kill.
	if t.cmd.Process != nil {
		t.cmd.Process.Kill()
	}

	<-t.done

	if t.cancel != nil {
		t.cancel()
	}

	return nil
}
