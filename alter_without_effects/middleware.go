package alter_without_effects

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	icontext "github.com/louvri/gokrt/context"
)

func Middleware(postprocessor func(original, data any, err error) (any, error), middlewares ...endpoint.Middleware) endpoint.Middleware {

	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req any) (any, error) {
			var ok bool
			var ictx *icontext.Context
			if ictx, ok = ctx.(*icontext.Context); !ok {
				ictx = icontext.New(ctx).(*icontext.Context)
			}
			outer := func(ctx context.Context, req any) (any, error) {
				return next(ctx, req)
			}
			curr := outer
			for i := len(middlewares) - 1; i >= 0; i-- {
				curr = middlewares[i](curr)
			}
			response, err := curr(ictx, req)
			if postprocessor != nil {
				return postprocessor(req, response, err)
			} else {
				return response, err
			}
		}
	}
}
