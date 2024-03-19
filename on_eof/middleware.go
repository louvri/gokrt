package on_eof

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/louvri/gokrt/sys_key"
)

func Middleware(middlewares ...endpoint.Middleware) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			eof := ctx.Value(sys_key.EOF)
			if eof != nil && eof == "eof" {
				for i := len(middlewares) - 1; i >= 0; i-- { // reverse
					next = middlewares[i](func(ctx context.Context, req interface{}) (interface{}, error) {
						return "", nil
					})
				}
				return next(ctx, req)
			}
			return next(ctx, req)
		}
	}
}
