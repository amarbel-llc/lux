package transport

import (
	"io"

	"github.com/friedenberg/lux/internal/jsonrpc"
)

type Stdio struct {
	stream *jsonrpc.Stream
	closer io.Closer
}

func NewStdio(r io.Reader, w io.Writer) *Stdio {
	return &Stdio{
		stream: jsonrpc.NewStream(r, w),
	}
}

func NewStdioWithCloser(r io.Reader, w io.Writer, c io.Closer) *Stdio {
	t := NewStdio(r, w)
	t.closer = c
	return t
}

func (t *Stdio) Read() (*jsonrpc.Message, error) {
	return t.stream.Read()
}

func (t *Stdio) Write(msg *jsonrpc.Message) error {
	return t.stream.Write(msg)
}

func (t *Stdio) Close() error {
	if t.closer != nil {
		return t.closer.Close()
	}
	return nil
}
