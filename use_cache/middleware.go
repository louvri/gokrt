package use_cache

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/louvri/gokrt/sys_key"
)

var CACHE_KEY = "cache_data"

func Middleware(e endpoint.Endpoint, preprocessor func(cache interface{}, next interface{}) interface{}, cacheKey ...string) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			key := CACHE_KEY
			if len(cacheKey) > 0 && cacheKey[0] != "" {
				key = cacheKey[0]
			}
			response, err := next(ctx, req)
			cache := ctx.Value(sys_key.CACHE_KEY)
			if cache != nil && err == nil {
				if cached, ok := cache.(map[string]interface{}); ok && cached[key] != nil {
					cache = cached[key]
				}
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
