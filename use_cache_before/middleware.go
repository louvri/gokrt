package use_cache_before

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/louvri/gokrt/sys_key"
)

func Middleware(processor func(cache interface{}, next interface{}) interface{}, cacheKey ...string) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			cache := ctx.Value(sys_key.CACHE_KEY)
			var key string
			if len(cacheKey) > 0 && cacheKey[0] != "" {
				key = cacheKey[0]
			}
			if cache != nil {
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
				alter := processor(tobeProcessed, req)
				if req != nil {
					_, err := next(ctx, alter)
					if err != nil {
						return nil, err
					}
				}
			}
			return nil, nil
		}
	}
}
