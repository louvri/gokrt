package context

import (
	"context"
	"time"

	"github.com/louvri/gokrt/sys_key"
)

var (
	Ictx interface{}
)

type ContextWithDeadline struct {
	base       context.Context
	properties Property
}

func New(ctx context.Context) IContext {
	if c, ok := ctx.(*ContextWithDeadline); ok {
		return c
	} else {
		return Hijack(ctx)
	}
}

func Hijack(ctx context.Context) *ContextWithDeadline {
	var base *ContextWithDeadline
	var properties Property
	properties = InitiateProperty(ctx)
	if tmp, ok := ctx.(*ContextWithDeadline); ok {
		base = tmp
	} else {
		base = &ContextWithDeadline{
			base:       ctx,
			properties: properties,
		}
	}
	return base

}

func (c *ContextWithDeadline) Get(key sys_key.SysKey) any {
	return c.properties.Get(key)
}

func (c *ContextWithDeadline) Set(key, value any) {
	if _, ok := key.(sys_key.SysKey); ok {
		c.properties.Set(key, value)
	} else {
		ctx := c.base
		ctx = context.WithValue(ctx, key, value)
		c.base = ctx
	}
}

// override
func (c *ContextWithDeadline) Deadline() (time.Time, bool) {
	return c.base.Deadline()
}

func (c *ContextWithDeadline) Done() <-chan struct{} { return c.base.Done() }

func (c *ContextWithDeadline) Value(key any) any {
	return c.base.Value(key)
}

func (c *ContextWithDeadline) Err() error {
	return c.base.Err()
}

func (c *ContextWithDeadline) WithoutDeadline(ctx context.Context) IContext {
	if current, ok := ctx.(*ContextWithoutDeadline); ok {
		return current
	}
	return NewContextWithoutDeadline(c.base, c.properties)
}

func (c *ContextWithDeadline) Base() context.Context {
	return c.base
}
