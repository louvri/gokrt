package cache

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/louvri/gokrt/sys_key"
)

func Middleware(e endpoint.Endpoint, preprocessor func(req interface{}) interface{}) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			if ctx.Value(sys_key.CACHE_KEY) == nil {
				response, err := e(ctx, preprocessor(req))
				if err != nil {
					return nil, err
				}
				ctx = context.WithValue(ctx, sys_key.CACHE_KEY, response)
			}
			return next(ctx, req)
		}
	}
}
