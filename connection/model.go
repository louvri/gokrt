package connection

import (
	"context"
	"io"
)

type Connection interface {
	Connect(ctx context.Context) error
	Cancel()
	Close()
	Writer() io.Writer
	Reader() io.Reader
	Handler() any
	Name() string
	Driver() string
}
