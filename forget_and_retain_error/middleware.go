package forget_and_retain_error

import (
	"context"

	"github.com/go-kit/kit/endpoint"
)

func Middleware(middlewares ...endpoint.Middleware) endpoint.Middleware {
	type cache struct {
		response any
		err      error
	}
	var c cache
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req any) (any, error) {
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
			_, err := curr(ctx, req)
			return c.response, err
		}
	}
}
