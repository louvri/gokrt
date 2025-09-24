package if_flag

import (
	"context"
	"sync"

	"github.com/go-kit/kit/endpoint"
	icontext "github.com/louvri/gokrt/context"
	"github.com/louvri/gokrt/sys_key"
	goRedis "github.com/redis/go-redis/v9"
)

func Middleware(key, field string, value any, redis *goRedis.Client, e endpoint.Endpoint, preprocessor func(data any, err error) any, wait ...bool) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req any) (any, error) {
			cmd := redis.HGet(context.Background(), key, field)
			curr := cmd.Val()
			if cmd.Err() != nil {
				return nil, cmd.Err()
			}
			if _, ok := ctx.Value(sys_key.GOKRT_CONTEXT).(*icontext.Context); !ok {
				ctx = icontext.New(ctx)
			}
			resp, err := next(ctx, req)
			if curr == value {
				if resp != nil || err != nil {
					result := preprocessor(resp, err)
					if result != nil {
						nwait := len(wait)
						if nwait > 0 && wait[0] {
							var wg sync.WaitGroup
							wg.Add(1)
							go func() {
								defer wg.Done()
								e(ctx, result)
							}()
							wg.Wait()
						} else {
							go e(ctx, result)
						}
					}
				}
			}
			return resp, err
		}
	}
}
