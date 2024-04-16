package flag

import (
	"context"
	"errors"
	"time"

	"github.com/go-kit/kit/endpoint"
	"github.com/louvri/gokrt/sys_key"
	goRedis "github.com/redis/go-redis/v9"
)

func Middleware(key, field string, value interface{}, duration time.Duration, redis *goRedis.Client) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			if value == nil {
				cmd := redis.HDel(ctx, key, field)
				if cmd.Err() != nil {
					return nil, cmd.Err()
				}
			} else {
				cmd := redis.HGet(ctx, key, field)
				if cmd.Err() != nil {
					if !errors.Is(cmd.Err(), goRedis.Nil) {
						return nil, cmd.Err()
					} else {
						redis.Expire(ctx, key, duration)
						redis.HSet(ctx, key, field, value)
					}
				} else {
					ctx = context.WithValue(ctx, sys_key.EOF, "err")
					return next(ctx, req)
				}
			}
			return next(ctx, req)
		}
	}
}
