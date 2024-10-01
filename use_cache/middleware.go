package use_cache

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/louvri/gokrt/option"
	"github.com/louvri/gokrt/sys_key"
)

func Middleware(e endpoint.Endpoint, preprocessor func(cache interface{}, next interface{}) interface{}, cacheConfig ...option.Config) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req interface{}) (interface{}, error) {

			config := map[option.Option]bool{}
			var key string

			if len(cacheConfig) > 0 {
				if len(cacheConfig[0].Option) > 0 {
					for _, opt := range cacheConfig[0].Option {
						if opt == option.EXECUTE_BEFORE {
							config[option.EXECUTE_BEFORE] = true
						}
					}
				}
				if cacheConfig[0].CacheKey != "" {
					key = cacheConfig[0].CacheKey
				}
			}

			var response interface{}
			var err error
			cache := ctx.Value(sys_key.CACHE_KEY)

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
				if req != nil {
					if config[option.EXECUTE_BEFORE] {
						var curr interface{}
						if e != nil {
							if curr, err = e(ctx, req); err != nil {
								return nil, err
							}
						}

						req = preprocessor(tobeProcessed, curr)
						if response, err = next(ctx, req); err != nil {
							return nil, err
						}
					} else {
						if response, err = next(ctx, req); err != nil {
							return nil, err
						}
						req = preprocessor(tobeProcessed, response)
						if e != nil {
							if _, err = e(ctx, req); err != nil {
								return nil, err
							}

						}
					}
				}
			}
			return response, err
		}
	}
}
