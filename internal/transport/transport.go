package transport

import (
	"github.com/amarbel-llc/go-lib-mcp/jsonrpc"
)

type Transport interface {
	Read() (*jsonrpc.Message, error)
	Write(*jsonrpc.Message) error
	Close() error
}
