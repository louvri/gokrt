package alter_without_effects

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	icontext "github.com/louvri/gokrt/context"
)

func Middleware(postprocessor func(original, data any, err error) (any, error), outer endpoint.Middleware, middlewares ...endpoint.Middleware) endpoint.Middleware {

	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req any) (any, error) {
			var ictx icontext.Icontext
			if tmp, ok := ctx.(icontext.Icontext); ok {
				ictx = tmp
			} else {
				ictx = icontext.New(ctx)
			}
			for i := len(middlewares) - 1; i >= 0; i-- {
				next = middlewares[i](next)
			}
			response, err := outer(next)(ictx, req)
			if postprocessor != nil {
				return postprocessor(req, response, err)
			} else {
				return response, err
			}
		}
	}
}
