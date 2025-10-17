package retry

import (
	"context"
	"time"

	"github.com/go-kit/kit/endpoint"
	"github.com/johnjerrico/hantu"
	"github.com/johnjerrico/hantu/schema"
	icontext "github.com/louvri/gokrt/context"
	"github.com/louvri/gokrt/sys_key"
)

func Middleware(id string, numberOfRetries int, waitTime time.Duration, onErrorMessage string, callback func(id string, request any, timestamp string), middlewares ...endpoint.Middleware) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		n := len(middlewares) - 1
		n_minus_one := n - 1
		for i := n; i >= 0; i-- { // reverse
			next = middlewares[i](next)
		}
		bg := hantu.Singleton(hantu.Option{
			Max: 50,
		})
		return func(ctx context.Context, request any) (any, error) {
			var ok bool
			var ictx *icontext.Context
			if ictx, ok = ctx.Value(sys_key.GOKRT_CONTEXT).(*icontext.Context); !ok {
				ictx = icontext.New(ctx).(*icontext.Context)
			}
			response, ierr := next(ictx, request)
			if ierr != nil && ierr.Error() == onErrorMessage {
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
				retry := middlewares[n](func(ctx context.Context, request any) (any, error) {
					return response, ierr
				})
				for i := n_minus_one; i >= 0; i-- {
					retry = middlewares[i](retry)
				}
				bg.Worker().Register(id, func(ctx context.Context, request any) {
					converted, ok := request.(map[string]any)
					if !ok {
						return
					}
					if callback != nil {
						callback(id, converted, timestamp)
					}
					_, err := retry(ictx.WithoutDeadline(), converted["request"])
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
			return response, ierr
		}
	}
}
