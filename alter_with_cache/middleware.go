package alter_with_cache

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/louvri/gokrt/sys_key"
)

func Middleware(e endpoint.Endpoint, preprocessor func(cache interface{}, next interface{}) interface{}, key ...string) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			var cache, response interface{}
			var err error
			cache = ctx.Value(sys_key.CACHE_KEY)
			id := ""
			if len(key) > 0 {
				id = key[0]
			}
			if id != "" {
				if cache != nil {
					if tmp, ok := cache.(map[string]interface{}); ok {
						if exist, ok := tmp[id]; ok {
							cache = exist
						}
					}
				}
			}
			req = preprocessor(cache, req)
			response, err = next(ctx, req)
			if err != nil {
				return response, err
			}
			return e(ctx, response)
		}
	}
}
