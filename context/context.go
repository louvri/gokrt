package context

import (
	"context"
	"time"

	"github.com/louvri/gokrt/sys_key"
)

var (
	Ictx interface{}
)

type Context struct {
	base       context.Context
	properties Property
}

func New(ctx context.Context) Icontext {
	if c, ok := ctx.(*Context); ok {
		return c
	} else {
		return Hijack(ctx)
	}
}

func Hijack(ctx context.Context) *Context {
	var base *Context
	var properties Property
	properties = InitiateProperty(ctx)
	if tmp, ok := ctx.(*Context); ok {
		base = tmp
	} else {
		base = &Context{
			base:       ctx,
			properties: properties,
		}
	}
	return base

}

func (c *Context) Get(key sys_key.SysKey) any {
	return c.properties.Get(key)
}

func (c *Context) Set(key, value any) {
	if _, ok := key.(sys_key.SysKey); ok {
		c.properties.Set(key, value)
	} else {
		ctx := c.base
		ctx = context.WithValue(ctx, key, value)
		c.base = ctx
	}
}

// override
func (c *Context) Deadline() (time.Time, bool) {
	return c.base.Deadline()
}

func (c *Context) Done() <-chan struct{} { return c.base.Done() }

func (c *Context) Value(key any) any {
	return c.base.Value(key)
}

func (c *Context) Err() error {
	return c.base.Err()
}

func (c *Context) WithoutDeadline() Icontext {
	return NewContextWithoutDeadline(c.base, c.properties)
}

func (c *Context) Base() context.Context {
	return c.base
}
