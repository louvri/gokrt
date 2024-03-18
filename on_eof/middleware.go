package on_eof

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/louvri/gokrt/sys_key"
)

func Middleware(middlewares ...endpoint.Middleware) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			outer := func(ctx context.Context, req interface{}) (interface{}, error) {
				resp, err := next(ctx, req)
				return resp, err
			}
			curr := outer
			for i := len(middlewares) - 1; i >= 0; i-- {
				curr = middlewares[i](curr)

			}
			eof := ctx.Value(sys_key.EOF)
			if eof != nil && eof == "eof" {
				return curr(ctx, req)
			}
			return next(ctx, req)
		}
	}
}
