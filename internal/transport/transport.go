package transport

import (
	"github.com/friedenberg/lux/internal/jsonrpc"
)

type Transport interface {
	Read() (*jsonrpc.Message, error)
	Write(*jsonrpc.Message) error
	Close() error
}
