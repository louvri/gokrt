package cache

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	icontext "github.com/louvri/gokrt/context"
	"github.com/louvri/gokrt/sys_key"
)

func Middleware(e endpoint.Endpoint, preprocessor func(req interface{}) interface{}, key ...string) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			var ictx *icontext.Context
			var ok bool
			if _, ok = ctx.Value(sys_key.INTERNAL_CONTEXT).(*icontext.Context); !ok {
				ctx = icontext.New(ctx)
			}

			ictx = ctx.Value(sys_key.INTERNAL_CONTEXT).(*icontext.Context)
			_cacheFromContext := ictx.Get(sys_key.CACHE_KEY)

			_key := ""
			if len(key) > 0 {
				_key = key[0]
			}
			iReq := req
			if preprocessor != nil {
				iReq = preprocessor(iReq)
			}
			if iReq != nil {
				data, err := e(ctx, iReq)
				if err != nil {
					return nil, err
				}
				var ok bool
				var tobeCached interface{}
				var cache map[string]interface{}
				if cache, ok = _cacheFromContext.(map[string]interface{}); ok {
					cache[_key] = data
					tobeCached = cache
				} else {
					if _key != "" {
						cache = make(map[string]interface{})
						cache[_key] = data
						tobeCached = cache
					} else {
						tobeCached = data
					}

				}
				// ctx = context.WithValue(ctx, sys_key.CACHE_KEY, tobeCached)
				ictx.Set(sys_key.CACHE_KEY, tobeCached)
			}
			return next(ctx, req)
		}
	}
}
