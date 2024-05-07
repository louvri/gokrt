package icontext

import "context"

type CopyContext struct {
	context.Context
	values context.Context
}

func New(baseContext, requestContext context.Context) context.Context {
	return &CopyContext{
		context: baseContext,
		values:  requestContext,
	}
}

func (c *CopyContext) Value(key any) any {
	return c.values.Value(key)
}
