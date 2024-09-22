package cache

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/louvri/gokrt/sys_key"
)

var CACHE_KEY = "cache_data"

func Middleware(e endpoint.Endpoint, preprocessor func(req interface{}) interface{}, cacheKey ...string) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			key := CACHE_KEY
			cached := make(map[string]interface{})
			if len(cacheKey) > 0 && cacheKey[0] != "" {
				key = cacheKey[0]
			}
			response, err := e(ctx, preprocessor(req))
			if err != nil {
				return nil, err
			}
			if ctx.Value(sys_key.CACHE_KEY) == nil {
				cached[key] = response
				ctx = context.WithValue(ctx, sys_key.CACHE_KEY, cached)
			} else {
				if cached, ok := ctx.Value(sys_key.CACHE_KEY).(map[string]interface{}); ok && len(cached) > 0 {
					cached[key] = response
					ctx = context.WithValue(ctx, sys_key.CACHE_KEY, cached)
				}
			}
			return next(ctx, req)
		}
	}
}
