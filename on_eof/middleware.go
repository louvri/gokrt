package on_eof

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/louvri/gokrt/sys_key"
)

func Middleware(middlewares ...endpoint.Middleware) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req any) (any, error) {
			eof := ctx.Value(sys_key.EOF)
			if eof != nil {
				next = func(ctx context.Context, req any) (any, error) {
					return "", nil
				}
				for i := len(middlewares) - 1; i >= 0; i-- { // reverse
					next = middlewares[i](next)
				}
				return next(ctx, req)
			}
			return next(ctx, req)
		}
	}
}
