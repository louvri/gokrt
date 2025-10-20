package forget

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	icontext "github.com/louvri/gokrt/context"
)

func Middleware(middlewares ...endpoint.Middleware) endpoint.Middleware {
	type cache struct {
		response any
		err      error
	}
	var c cache
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req any) (any, error) {
			var ok bool
			var ictx *icontext.Context
			if ictx, ok = ctx.(*icontext.Context); !ok {
				ictx = icontext.New(ctx).(*icontext.Context)
			}
			outer := func(ctx context.Context, req any) (any, error) {
				resp, err := next(ctx, req)
				c.response = resp
				c.err = err
				return resp, err
			}
			curr := outer
			for i := len(middlewares) - 1; i >= 0; i-- {
				curr = middlewares[i](curr)
			}
			curr(ictx, req)
			return c.response, c.err
		}
	}
}
