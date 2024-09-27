package cache

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/louvri/gokrt/sys_key"
)

func Middleware(e endpoint.Endpoint, preprocessor func(req interface{}) interface{}, CACHE_KEY_STR ...string) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			existingCache := ctx.Value(sys_key.CACHE_KEY)
			var key string
			if len(CACHE_KEY_STR) > 0 && CACHE_KEY_STR[0] != "" {
				key = CACHE_KEY_STR[0]
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
				} else if !ok {
					tobeCached = map[string]interface{}{
						"": existingCache,
					}
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
