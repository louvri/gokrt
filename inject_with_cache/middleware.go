package inject_with_cache

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/louvri/gokrt/option"
	"github.com/louvri/gokrt/sys_key"
)

func Middleware(preprocessor func(cache, data interface{}) interface{}, cacheOption ...option.Config) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			var cache interface{}

			cache = ctx.Value(sys_key.CACHE_KEY)
			config := map[option.Option]bool{}
			key := ""
			if len(cacheOption) > 0 {
				if cacheOption[0].CacheKey != "" {
					key = cacheOption[0].CacheKey
				}
				if len(cacheOption[0].Option) > 0 {
					for _, opt := range cacheOption[0].Option {
						config[opt] = true
					}
				}
			}

			if key != "" {
				if cache != nil {
					if tmp, ok := cache.(map[string]interface{}); ok {
						if exist, ok := tmp[key]; ok {
							cache = exist
						}
					}
				}
			}
			req = preprocessor(cache, req)
			return next(ctx, req)
		}
	}
}
