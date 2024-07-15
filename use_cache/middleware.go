package use_cache

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/louvri/gokrt/sys_key"
)

func Middleware(e endpoint.Endpoint, preprocessor func(cache interface{}, next interface{}) interface{}) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			response, err := next(ctx, req)
			cache := ctx.Value(sys_key.CACHE_KEY)
			if cache != nil && err == nil {
				req = preprocessor(cache, response)
				if req != nil {
					_, err := e(ctx, req)
					if err != nil {
						return nil, err
					}
				}
			}
			return response, err
		}
	}
}
