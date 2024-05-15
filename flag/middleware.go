package flag

import (
	"context"
	"errors"
	"time"

	"github.com/go-kit/kit/endpoint"
	"github.com/louvri/gokrt/sys_key"
	goRedis "github.com/redis/go-redis/v9"
)

func Middleware(key, field string, value, endstate interface{}, duration time.Duration, redis *goRedis.Client) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			if value == nil {
				cmd := redis.HDel(ctx, key, field)
				if cmd.Err() != nil {
					if !errors.Is(cmd.Err(), goRedis.Nil) {
						ctx = context.WithValue(ctx, sys_key.EOF, "err")
						return next(ctx, req)
					}
				}
			} else {
				cmd := redis.HGet(ctx, key, field)
				curr := cmd.Val()
				if cmd.Err() != nil {
					if !errors.Is(cmd.Err(), goRedis.Nil) {
						ctx = context.WithValue(ctx, sys_key.EOF, "err")
						return next(ctx, req)
					}
				} else if curr == endstate {
					ctx = context.WithValue(ctx, sys_key.EOF, curr)
					return next(ctx, req)
				}
			}
			response, err := next(ctx, req)
			redis.Expire(ctx, key, duration)
			redis.HSet(ctx, key, field, value)
			return response, err
		}
	}
}
func Last(key, field string, redis *goRedis.Client) (interface{}, error) {
	cmd := redis.HGet(context.Background(), key, field)
	curr := cmd.Val()
	if cmd.Err() != nil {
		return nil, cmd.Err()
	}
	return curr, nil
}
