package flag

import (
	"context"
	"errors"
	"time"

	"github.com/go-kit/kit/endpoint"
	icontext "github.com/louvri/gokrt/context"
	"github.com/louvri/gokrt/sys_key"
	goRedis "github.com/redis/go-redis/v9"
)

func Middleware(key, field string, value, endstate any, duration time.Duration, redis *goRedis.Client) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req any) (any, error) {
			var ok bool
			var ictx *icontext.Context
			if ictx, ok = ctx.Value(sys_key.GOKRT_CONTEXT).(*icontext.Context); !ok {
				ictx = icontext.New(ctx).(*icontext.Context)
			}
			if value == nil {
				cmd := redis.HDel(ictx, key, field)
				if cmd.Err() != nil {
					if !errors.Is(cmd.Err(), goRedis.Nil) {
						ictx.Set(sys_key.EOF, "err")
						// ctx = context.WithValue(ctx, sys_key.EOF, "err")
						return next(ictx, req)
					}
				}
			} else {
				cmd := redis.HGet(ictx, key, field)
				curr := cmd.Val()
				if cmd.Err() != nil {
					if !errors.Is(cmd.Err(), goRedis.Nil) {
						ictx.Set(sys_key.EOF, "err")
						// ctx = context.WithValue(ctx, sys_key.EOF, "err")
						return next(ctx, req)
					}
				} else if curr == endstate {
					ictx.Set(sys_key.EOF, curr)
					// ctx = context.WithValue(ctx, sys_key.EOF, curr)
					return next(ctx, req)
				}
			}
			redis.Expire(ctx, key, duration)
			redis.HSet(ctx, key, field, value)
			return next(ctx, req)
		}
	}
}
