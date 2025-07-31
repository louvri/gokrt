package forget

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	icontext "github.com/louvri/gokrt/context"
	"github.com/louvri/gokrt/sys_key"
)

func Middleware(middlewares ...endpoint.Middleware) endpoint.Middleware {
	type cache struct {
		response any
		err      error
	}
	var c cache
	return func(next endpoint.Endpoint) endpoint.Endpoint {
<<<<<<< HEAD
		return func(ctx context.Context, req any) (any, error) {
			if _, ok := ctx.Value(sys_key.INTERNAL_CONTEXT).(*icontext.Context); !ok {
				ctx = icontext.New(ctx)
			}
			outer := func(ctx context.Context, req any) (any, error) {
=======
		return func(ctx context.Context, req any) (any, error) {
			outer := func(ctx context.Context, req any) (any, error) {
>>>>>>> main
				resp, err := next(ctx, req)
				c.response = resp
				c.err = err
				return resp, err
			}
			curr := outer
			for i := len(middlewares) - 1; i >= 0; i-- {
				curr = middlewares[i](curr)
			}
			curr(ctx, req)
			return c.response, c.err
		}
	}
}
