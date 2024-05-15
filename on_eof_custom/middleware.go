package on_eof_custom

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/louvri/gokrt/sys_key"
)

func Middleware(condition func(ctx context.Context) bool, deferFunction func(ctx context.Context) error, middlewares ...endpoint.Middleware) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			eof := ctx.Value(sys_key.EOF)
			if eof != nil && eof == "eof" && condition(ctx) {
				next = func(ctx context.Context, req interface{}) (interface{}, error) {
					return "", nil
				}
				for i := len(middlewares) - 1; i >= 0; i-- { // reverse
					next = middlewares[i](next)
				}
				return next(ctx, req)
			} else if eof != nil {
				deferFunction(ctx)
				return nil, nil
			}
			return next(ctx, req)
		}
	}
}
