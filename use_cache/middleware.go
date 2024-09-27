package use_cache

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/louvri/gokrt/sys_key"
)

func Middleware(e endpoint.Endpoint, preprocessor func(cache interface{}, next interface{}) interface{}, CACHE_KEY_STR ...string) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			response, err := next(ctx, req)
			cache := ctx.Value(sys_key.CACHE_KEY)
			var key string
			if len(CACHE_KEY_STR) > 0 && CACHE_KEY_STR[0] != "" {
				key = CACHE_KEY_STR[0]
			}
			if cache != nil && err == nil {
				var tobeProcessed interface{}
				tobeProcessed = cache
				if exist, ok := cache.(map[string]interface{}); ok {
					if key == "" {
						tobeProcessed = exist
					} else if key != "" {
						if tmp, ok := exist[key]; ok {
							tobeProcessed = tmp
						}
					}
				}
				req = preprocessor(tobeProcessed, response)
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
