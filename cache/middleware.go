package cache

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/louvri/gokrt/option"
	"github.com/louvri/gokrt/sys_key"
)

func Middleware(e endpoint.Endpoint, preprocessor func(req interface{}) interface{}, cacheConfig ...option.Config) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			existingCache := ctx.Value(sys_key.CACHE_KEY)
			var key string
			config := map[option.Option]bool{}
			if len(cacheConfig) > 0 {
				key = cacheConfig[0].CacheKey
				if len(cacheConfig[0].Option) > 0 {
					for _, opt := range cacheConfig[0].Option {
						if opt == option.EXECUTE_BEFORE {
							config[option.EXECUTE_BEFORE] = true
						}

						if opt == option.EXECUTE_AFTER {
							config[option.EXECUTE_AFTER] = true
						}
					}
				}
			}

			if existingCache == nil && key == "" {
				response, err := e(ctx, preprocessor(req))
				if err != nil {
					return nil, err
				}
				ctx = context.WithValue(ctx, sys_key.CACHE_KEY, response)
			} else if existingCache == nil && key != "" {
				response, err := e(ctx, preprocessor(req))
				if err != nil {
					return nil, err
				}
				ctx = context.WithValue(ctx, sys_key.CACHE_KEY, map[string]interface{}{
					key: response,
				})
			} else if existingCache != nil {
				tobeCached := make(map[string]interface{})
				if mapExist, ok := existingCache.(map[string]interface{}); ok {
					tobeCached = mapExist
				}

				response, err := e(ctx, preprocessor(req))
				if err != nil {
					return nil, err
				}
				if key != "" {
					tobeCached[key] = response
					ctx = context.WithValue(ctx, sys_key.CACHE_KEY, tobeCached)
				} else if key == "" {
					ctx = context.WithValue(ctx, sys_key.CACHE_KEY, response)
				}
			}
			return next(ctx, req)
		}
	}
}
