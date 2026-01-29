package icontext

import (
	"context"
	"time"

	"github.com/louvri/gokrt/sys_key"
)

type Context struct {
	base       context.Context
	properties map[sys_key.SysKey]any
}

func New(ctx context.Context) context.Context {
	if _, ok := ctx.(*Context); ok {
		return ctx
	} else {
		return Hijack(ctx)
	}
}

func Hijack(ctx context.Context) *Context {
	var base *Context
	if tmp, ok := ctx.(*Context); ok {
		base = tmp
	} else {
		base = &Context{
			base: ctx,
			properties: map[sys_key.SysKey]any{
				sys_key.FILE_KEY:        ctx.Value(sys_key.FILE_KEY),
				sys_key.FILE_OBJECT_KEY: ctx.Value(sys_key.FILE_OBJECT_KEY),
				sys_key.SOF:             ctx.Value(sys_key.SOF),
				sys_key.EOF:             ctx.Value(sys_key.EOF),
				sys_key.DATA_REF:        ctx.Value(sys_key.DATA_REF),
				sys_key.CACHE_KEY:       ctx.Value(sys_key.CACHE_KEY),
			},
		}
	}
	return base

}

func (c *Context) Get(key sys_key.SysKey) any {
	switch key {
	case sys_key.FILE_KEY:
		return c.properties[sys_key.FILE_KEY]
	case sys_key.FILE_OBJECT_KEY:
		return c.properties[sys_key.FILE_OBJECT_KEY]
	case sys_key.SOF:
		return c.properties[sys_key.SOF]
	case sys_key.EOF:
		return c.properties[sys_key.EOF]
	case sys_key.DATA_REF:
		return c.properties[sys_key.DATA_REF]
	case sys_key.CACHE_KEY:
		return c.properties[sys_key.CACHE_KEY]
	case sys_key.GOKRT_CONTEXT:
		return c.properties[sys_key.GOKRT_CONTEXT]
	default:
		return c.base.Value(key)
	}
}

func (c *Context) Set(key, value any) {
	switch key {
	case sys_key.FILE_KEY:
		c.properties[sys_key.FILE_KEY] = value
	case sys_key.FILE_OBJECT_KEY:
		c.properties[sys_key.FILE_OBJECT_KEY] = value
	case sys_key.SOF:
		c.properties[sys_key.SOF] = value
	case sys_key.EOF:
		c.properties[sys_key.EOF] = value
	case sys_key.DATA_REF:
		c.properties[sys_key.DATA_REF] = value
	case sys_key.CACHE_KEY:
		c.properties[sys_key.CACHE_KEY] = value
	case sys_key.GOKRT_CONTEXT:
		c.properties[sys_key.GOKRT_CONTEXT] = value
	default:
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

func (c *Context) WithoutDeadline() context.Context {
	return NewContextWithoutDeadline(c.base)
}
