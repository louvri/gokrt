package use_cache

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/louvri/gokrt/sys_key"
)

var CACHE_KEY = "cache_data"

func Middleware(e endpoint.Endpoint, preprocessor func(cache interface{}, next interface{}) interface{}, MULTIPLE_CACHE_STORED ...bool) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			response, err := next(ctx, req)
			cache := ctx.Value(sys_key.CACHE_KEY)
			if cache != nil && err == nil {
				if cached, ok := cache.(map[string]interface{}); ok {
					if len(MULTIPLE_CACHE_STORED) > 0 && MULTIPLE_CACHE_STORED[0] {
						cache = cached
					} else {
						// let the middleware know if the cache is stored on multipled or not
						cache = cached[CACHE_KEY]
					}

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
