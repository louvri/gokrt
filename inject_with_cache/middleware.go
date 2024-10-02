package inject_with_cache

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/louvri/gokrt/option"
	"github.com/louvri/gokrt/sys_key"
)

func Middleware(e endpoint.Endpoint, preprocessor func(cache, data interface{}) interface{}, postprocessor func(cache, original, data interface{}, err error) interface{}, cacheOption ...option.Config) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			var cache, response, curr interface{}
			var err error
			cache = ctx.Value(sys_key.CACHE_KEY)

			key := ""
			if len(cacheOption) > 0 {
				if cacheOption[0].CacheKey != "" {
					key = cacheOption[0].CacheKey
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

			curr = preprocessor(cache, req)
			if e != nil {
				response, err = e(ctx, curr)
			}
			if err != nil {
				return nil, err
			}

			req = postprocessor(cache, curr, response, err)
			return next(ctx, req)
		}
	}
}
