package on_eof

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	icontext "github.com/louvri/gokrt/context"
	"github.com/louvri/gokrt/sys_key"
)

func Middleware(middlewares ...endpoint.Middleware) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req any) (any, error) {

			var ictx *icontext.Context

			if _, ok := ctx.Value(sys_key.GOKRT_CONTEXT).(*icontext.Context); !ok {
				ctx = icontext.New(ctx)
			}
			ictx = ctx.Value(sys_key.GOKRT_CONTEXT).(*icontext.Context)
			eof := ictx.Get(sys_key.EOF)

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
