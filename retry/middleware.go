package retry

import (
	"context"
	"time"

	"github.com/go-kit/kit/endpoint"
	"github.com/johnjerrico/hantu"
	"github.com/johnjerrico/hantu/schema"
)

func Middleware(id string, numberOfRetries int, waitTime time.Duration, outer endpoint.Middleware, others ...endpoint.Middleware) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		for i := len(others) - 1; i >= 0; i-- { // reverse
			next = others[i](next)
		}
		bg := hantu.Singleton(hantu.Option{
			Max: 50,
		})
		return func(ctx context.Context, request any) (any, error) {
			response, err := next(ctx, request)
			if err != nil {
				injectedReq := make(map[string]any)
				injectedReq["request"] = request
				injectedReq["counter"] = 0
				timestamp := time.Now().Format("2006-01-02 15:04:05")
				bg.Queue(schema.Job{
					Id:      timestamp,
					Name:    id,
					Ctx:     ctx,
					Request: injectedReq,
					Delay:   waitTime,
				})
				bg.Worker().Register(id, func(ctx context.Context, request any) {
					converted, ok := request.(map[string]any)
					if !ok {
						return
					}
					_, err := next(ctx, converted["request"])
					if err != nil {
						if cnt, ok := converted["counter"].(int); ok && cnt < numberOfRetries {
							converted["counter"] = cnt + 1
							bg.Queue(schema.Job{
								Id:      timestamp,
								Name:    id,
								Ctx:     ctx,
								Request: converted,
								Delay:   waitTime,
							})
						}
					}
				})
			}
			return response, err
		}
	}
}
