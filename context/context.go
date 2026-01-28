package icontext

import (
	"context"
	"reflect"
	"time"
	"unsafe"

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
				sys_key.FILE_KEY:                 ctx.Value(sys_key.FILE_KEY),
				sys_key.FILE_OBJECT_KEY:          ctx.Value(sys_key.FILE_OBJECT_KEY),
				sys_key.SOF:                      ctx.Value(sys_key.SOF),
				sys_key.EOF:                      ctx.Value(sys_key.EOF),
				sys_key.DATA_REF:                 ctx.Value(sys_key.DATA_REF),
				sys_key.CACHE_KEY:                ctx.Value(sys_key.CACHE_KEY),
				sys_key.GOKRT_CONTEXT_RESET_FLAG: false,
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
	// Unwrap to get the actual base
	baseCtx := c.base
	if _, ok := baseCtx.(*ContextWithoutDeadline); ok {
		return c
	}

	var altered context.Context
	if reset, ok := c.properties[sys_key.GOKRT_CONTEXT_RESET_FLAG].(bool); ok && !reset {
		existing := extract(baseCtx)
		altered = context.Background()

		for key, val := range existing {
			altered = context.WithValue(altered, key, val)
		}
		c.properties[sys_key.GOKRT_CONTEXT_RESET_FLAG] = true
	} else {
		altered = baseCtx
	}

	newCtx := &Context{
		base:       NewContextWithoutDeadline(altered),
		properties: make(map[sys_key.SysKey]any, len(c.properties)),
	}

	for k, v := range c.properties {
		newCtx.properties[k] = v
	}

	return newCtx
}

func extract(ctx context.Context) map[any]any {
	values := make(map[any]any)

	currentCtx := ctx
	for currentCtx != nil {
		// Get the reflect value
		val := reflect.ValueOf(currentCtx)

		// If it's a pointer, get the element
		if val.Kind() == reflect.Ptr {
			val = val.Elem()
		}

		// Check if it's a struct
		if val.Kind() != reflect.Struct {
			break
		}

		// Try to find "key" and "val" fields (valueCtx structure)
		keyField := val.FieldByName("key")
		valField := val.FieldByName("val")

		if keyField.IsValid() && valField.IsValid() {
			// Use unsafe to access unexported fields
			keyField = reflect.NewAt(keyField.Type(), unsafe.Pointer(keyField.UnsafeAddr())).Elem()
			valField = reflect.NewAt(valField.Type(), unsafe.Pointer(valField.UnsafeAddr())).Elem()

			if keyField.CanInterface() && valField.CanInterface() {
				values[keyField.Interface()] = valField.Interface()
			}
		}

		// Try to get parent context
		parentField := val.FieldByName("Context")
		if !parentField.IsValid() {
			break
		}

		// Use unsafe to access unexported field
		parentField = reflect.NewAt(parentField.Type(), unsafe.Pointer(parentField.UnsafeAddr())).Elem()

		if !parentField.CanInterface() {
			break
		}

		// Move to parent context
		if nextCtx, ok := parentField.Interface().(context.Context); ok {
			currentCtx = nextCtx
		} else {
			break
		}
	}

	return values
}
