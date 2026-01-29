package icontext

import (
	"context"
	"time"

	"github.com/louvri/gokrt/sys_key"
)

type Context interface {
	Deadline() (time.Time, bool)
	Done() <-chan struct{}
	Value(key any) any
	Err() error
	Get(key sys_key.SysKey) any
	Set(key, value any)
}
type icontext struct {
	base       context.Context
	properties map[sys_key.SysKey]any
}

func New(ctx context.Context) context.Context {
	if _, ok := ctx.(*icontext); ok {
		return ctx
	} else {
		return Hijack(ctx)
	}
}

func Hijack(ctx context.Context) Context {
	var base *icontext
	if tmp, ok := ctx.(*icontext); ok {
		base = tmp
	} else {
		base = &icontext{
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

func (c *icontext) Get(key sys_key.SysKey) any {
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
	case sys_key.DETACH_DEADLINE:
		return c.properties[sys_key.DETACH_DEADLINE]
	default:
		return c.base.Value(key)
	}
}

func (c *icontext) Set(key, value any) {
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
	case sys_key.DETACH_DEADLINE:
		c.properties[sys_key.DETACH_DEADLINE] = value
	default:
		ctx := c.base
		ctx = context.WithValue(ctx, key, value)
		c.base = ctx
	}
}

// override
func (c *icontext) Deadline() (time.Time, bool) {
	if detachDeadline, ok := c.properties[sys_key.DETACH_DEADLINE].(bool); ok && detachDeadline {
		return time.Time{}, false
	}
	return c.base.Deadline()
}

func (c *icontext) Done() <-chan struct{} {
	if detachDeadline, ok := c.properties[sys_key.DETACH_DEADLINE].(bool); ok && detachDeadline {
		return nil
	}
	return c.base.Done()
}

func (c *icontext) Value(key any) any {
	return c.base.Value(key)
}

func (c *icontext) Err() error {
	if detachDeadline, ok := c.properties[sys_key.DETACH_DEADLINE].(bool); ok && detachDeadline {
		return c.base.Err()
	}
	return c.base.Err()
}

// func (c *Context) WithoutDeadline() context.Context {

// 	if _, hasDeadline := c.base.Deadline(); !hasDeadline {
// 		return c
// 	}
// 	c.base = NewContextWithoutDeadline(c.base)
// 	return c
// }
