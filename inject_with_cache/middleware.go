package inject_with_cache

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	icontext "github.com/louvri/gokrt/context"
	"github.com/louvri/gokrt/sys_key"
)

func Middleware(preprocessor func(cache, data any) any, keys ...string) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req any) (any, error) {
			if _, ok := ctx.Value(sys_key.INTERNAL_CONTEXT).(*icontext.Context); !ok {
				ctx = icontext.New(ctx)
			}
			ictx, _ := ctx.Value(sys_key.INTERNAL_CONTEXT).(*icontext.Context)
			inmem := ictx.Get(sys_key.CACHE_KEY)
			if inmemCache, ok := inmem.(map[string]any); ok {
				key := ""
				nKey := len(keys)
				if nKey > 0 {
					if nKey == 1 {
						key = keys[0]
						req = preprocessor(inmemCache[key], req)
						return next(ctx, req)
					} else if nKey > 1 {
						new := make(map[string]any)
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
