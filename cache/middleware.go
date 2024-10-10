package cache

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/louvri/gokrt/sys_key"
)

func Middleware(e endpoint.Endpoint, preprocessor func(req interface{}) interface{}, key ...string) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			existingCache := ctx.Value(sys_key.CACHE_KEY)
			var id string
			if len(key) > 0 {
				id = key[0]
			}
			if existingCache == nil && id == "" {
				response, err := e(ctx, preprocessor(req))
				if err != nil {
					return nil, err
				}
				ctx = context.WithValue(ctx, sys_key.CACHE_KEY, response)
			} else if existingCache == nil && id != "" {
				response, err := e(ctx, preprocessor(req))
				if err != nil {
					return nil, err
				}
				ctx = context.WithValue(ctx, sys_key.CACHE_KEY, map[string]interface{}{
					id: response,
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
				if id != "" {
					tobeCached[id] = response
					ctx = context.WithValue(ctx, sys_key.CACHE_KEY, tobeCached)
				} else if id == "" {
					ctx = context.WithValue(ctx, sys_key.CACHE_KEY, response)
				}
			}
			return next(ctx, req)
		}
	}
}
