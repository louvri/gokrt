package icontext

import (
	"context"
	"time"
)

type ContextWithoutDeadline struct {
	base context.Context
}

func NewContextWithoutDeadline(ctx context.Context) context.Context {
	return &ContextWithoutDeadline{
		base: ctx,
	}
}
func (c *ContextWithoutDeadline) Base() context.Context {
	return c.base
}

func (c *ContextWithoutDeadline) Deadline() (time.Time, bool) {
	return time.Time{}, false
}

func (c *ContextWithoutDeadline) Done() <-chan struct{} { return nil }

func (c *ContextWithoutDeadline) Value(key any) any {
	return c.base.Value(key)

}
func (c *ContextWithoutDeadline) Err() error {
	return nil
}
