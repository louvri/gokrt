package if_flag

import (
	"context"
	"sync"

	"github.com/go-kit/kit/endpoint"
	icontext "github.com/louvri/gokrt/context"
	goRedis "github.com/redis/go-redis/v9"
)

func Middleware(key, field string, value any, redis *goRedis.Client, e endpoint.Endpoint, preprocessor func(data any, err error) any, wait ...bool) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req any) (any, error) {
			var ictx *icontext.Context

			if tmp, ok := ctx.(*icontext.Context); ok {
				ictx = tmp
			} else {
				ictx = icontext.New(ctx).(*icontext.Context)
			}
			cmd := redis.HGet(ictx, key, field)
			curr := cmd.Val()
			if cmd.Err() != nil {
				return nil, cmd.Err()
			}
			resp, err := next(ictx, req)
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
								e(ictx.WithoutDeadline(), result)
							}()
							wg.Wait()
						} else {
							go e(ictx.WithoutDeadline(), result)
						}
					}
				}
			}
			return resp, err
		}
	}
}
