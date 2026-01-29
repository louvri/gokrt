package context

import (
	"context"
	"time"

	"github.com/louvri/gokrt/sys_key"
)

type ContextWithoutDeadline struct {
	base       context.Context
	properties Property
}

func (c *ContextWithoutDeadline) Get(key sys_key.SysKey) any {
	return c.properties.Get(key)
}

func (c *ContextWithoutDeadline) Set(key, value any) {
	if _, ok := key.(sys_key.SysKey); ok {
		c.properties.Set(key, value)
	} else {
		ctx := c.base
		ctx = context.WithValue(ctx, key, value)
		c.base = ctx
	}
}

func NewContextWithoutDeadline(ctx context.Context, properties Property) *ContextWithoutDeadline {
	return &ContextWithoutDeadline{
		base:       ctx,
		properties: properties,
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

func (c *ContextWithoutDeadline) WithoutDeadline(ctx context.Context) IContext {
	return c
}
