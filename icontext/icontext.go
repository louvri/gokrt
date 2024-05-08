package icontext

import (
	"context"
	"strings"
	"time"
)

type CopyContext struct {
	ctx      context.Context
	deadline time.Time
}

func New(ctx context.Context, deadline time.Time) context.Context {
	return &CopyContext{
		ctx:      ctx,
		deadline: deadline,
	}
}

func (c *CopyContext) Deadline() (time.Time, bool) {
	if time.Now().After(c.deadline) {
		return c.deadline, true
	}
	return c.deadline, false
}
func (c *CopyContext) Done() <-chan struct{} { return nil }
func (c *CopyContext) Value(key interface{}) interface{} {
	return c.ctx.Value(key)
}
func (c *CopyContext) Err() error {
	if strings.Contains(c.ctx.Err().Error(), "deadline") ||
		strings.Contains(c.ctx.Err().Error(), "canceled") {
		return nil
	}
	return c.ctx.Err()
}
