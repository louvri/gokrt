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

						if opt == option.FORBID_CURRENT_ENDPOINT_RUN {
							config[option.FORBID_CURRENT_ENDPOINT_RUN] = true
						}
					}
				}
				if cacheConfig[0].CacheKey != "" {
					key = cacheConfig[0].CacheKey
				}
			}

			var response interface{}
			var err error

			if config[option.EXECUTE_BEFORE] && !config[option.FORBID_CURRENT_ENDPOINT_RUN] {
				response, err = e(ctx, req)
			} else if config[option.FORBID_CURRENT_ENDPOINT_RUN] {
				response = nil
				err = nil
			} else {
				response, err = next(ctx, req)
			}
			cache := ctx.Value(sys_key.CACHE_KEY)

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
					if config[option.EXECUTE_BEFORE] {
						_, err = next(ctx, req)
					} else {
						_, err = e(ctx, req)
					}
					if err != nil {
						return nil, err
					}
				}
			}
			return response, err
		}
	}
}
