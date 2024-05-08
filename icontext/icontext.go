package icontext

import (
	"context"
	"time"
)

type CopyContext struct {
	ctx context.Context
}

func New(ctx context.Context, deadline time.Time) context.Context {
	return &CopyContext{
		ctx: ctx,
	}
}

func (c *CopyContext) Deadline() (time.Time, bool) {
	return time.Time{}, false
}
func (c *CopyContext) Done() <-chan struct{} { return nil }
func (c *CopyContext) Value(key interface{}) interface{} {
	return c.ctx.Value(key)
}
func (c *CopyContext) Err() error {
	return nil
}
