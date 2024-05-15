package if_flag

import (
	"context"
	"sync"
	"time"

	"github.com/go-kit/kit/endpoint"
	"github.com/louvri/gokrt/icontext"
	goRedis "github.com/redis/go-redis/v9"
)

func Middleware(key, field string, value interface{}, redis *goRedis.Client, e endpoint.Endpoint, preprocessor func(data interface{}, err error) interface{}, wait ...bool) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			cmd := redis.HGet(context.Background(), key, field)
			curr := cmd.Val()
			if cmd.Err() != nil {
				return nil, cmd.Err()
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
								e(ctx, result)
								wg.Done()
							}()
							wg.Wait()
						} else {
							if _, ok := ctx.(*icontext.CopyContext); !ok {
								ctx = icontext.New(ctx, time.Now().Add(5*time.Second))
							}
							go e(ctx, result)
						}
					}
				}
			}
			return resp, err
		}
	}
}
