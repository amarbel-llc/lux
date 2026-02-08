package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"github.com/friedenberg/lux/internal/jsonrpc"
)

// StdioTransport implements MCP stdio transport using newline-delimited JSON.
// This differs from LSP which uses Content-Length headers.
type StdioTransport struct {
	scanner *bufio.Scanner
	writer  io.Writer
	closer  io.Closer
	mu      sync.Mutex
}

func NewStdioTransport(r io.Reader, w io.Writer) *StdioTransport {
	scanner := bufio.NewScanner(r)
	// Increase buffer size for large messages
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	return &StdioTransport{
		scanner: scanner,
		writer:  w,
	}
}

func NewStdioTransportWithCloser(r io.Reader, w io.Writer, c io.Closer) *StdioTransport {
	t := NewStdioTransport(r, w)
	t.closer = c
	return t
}

func (t *StdioTransport) Read() (*jsonrpc.Message, error) {
	if !t.scanner.Scan() {
		if err := t.scanner.Err(); err != nil {
			return nil, fmt.Errorf("reading message: %w", err)
		}
		return nil, io.EOF
	}

	line := t.scanner.Bytes()
	if len(line) == 0 {
		// Skip empty lines
		return t.Read()
	}

	var msg jsonrpc.Message
	if err := json.Unmarshal(line, &msg); err != nil {
		return nil, fmt.Errorf("parsing message: %w", err)
	}

	return &msg, nil
}

func (t *StdioTransport) Write(msg *jsonrpc.Message) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshaling message: %w", err)
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	if _, err := fmt.Fprintf(t.writer, "%s\n", data); err != nil {
		return fmt.Errorf("writing message: %w", err)
	}

	return nil
}

func (t *StdioTransport) Close() error {
	if t.closer != nil {
		return t.closer.Close()
	}
	return nil
}
