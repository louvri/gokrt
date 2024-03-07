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
	Handler() interface{}
	Name() string
	Driver() string
}
