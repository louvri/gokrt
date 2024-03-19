package on_eof

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/louvri/gokrt/sys_key"
)

func Middleware(outer endpoint.Middleware, others ...endpoint.Middleware) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			for i := len(others) - 1; i >= 0; i-- { // reverse
				next = others[i](next)
			}
			eof := ctx.Value(sys_key.EOF)
			if eof != nil && eof == "eof" {
				return outer(next)(ctx, req)
			}
			return next(ctx, req)
		}
	}
}
