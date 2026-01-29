package context

import (
	"context"
	"time"

	"github.com/louvri/gokrt/sys_key"
)

type IContext interface {
	context.Context
	Get(key sys_key.SysKey) any
	Set(key, value any)
	Base() context.Context
	Deadline() (time.Time, bool)
	Done() <-chan struct{}
	Value(key any) any
	Err() error
	WithoutDeadline(ctx context.Context) IContext
}
