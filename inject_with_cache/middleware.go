package inject_with_cache

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/louvri/gokrt/sys_key"
)

func Middleware(preprocessor func(cache, data interface{}) interface{}, keys ...string) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			inmem := ctx.Value(sys_key.CACHE_KEY)
			if inmemCache, ok := inmem.(map[string]interface{}); ok {
				key := ""
				nKey := len(keys)
				if nKey > 0 {
					if nKey == 1 {
						key = keys[0]
						req = preprocessor(inmemCache[key], req)
						return next(ctx, req)
					} else if nKey > 1 {
						new := make(map[string]interface{})
						for _, k := range keys {
							new[k] = inmemCache[key]
						}
						req := preprocessor(new, req)
						return next(ctx, req)
					}
				}
			}
			req = preprocessor(inmem, req)
			return next(ctx, req)
		}
	}
}
