package retry

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kit/kit/endpoint"
	"github.com/johnjerrico/hantu"
	"github.com/johnjerrico/hantu/schema"
	icontext "github.com/louvri/gokrt/context"
)

func Middleware(id string, numberOfRetries int, waitTime time.Duration, onErrorMessage string, callback func(id string, request any, timestamp string), middlewares ...endpoint.Middleware) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		n := len(middlewares) - 1
		for i := n; i >= 0; i-- {
			next = middlewares[i](next)
		}
		//intentionaly to skip the root call and go just run the middlewares
		retry := middlewares[n](func(ctx context.Context, request any) (any, error) {
			return nil, nil
		})
		for i := n - 1; i >= 0; i-- {
			retry = middlewares[i](retry)
		}
		bg := hantu.Singleton(hantu.Option{
			Max: 50,
		})
		workerName := fmt.Sprintf("%s:%p", id, &next)
		bg.Worker().Register(workerName, func(ctx context.Context, request any) {
			job, ok := request.(map[string]any)
			if !ok {
				return
			}
			if callback != nil {
				jobTimestamp := job["timestamp"].(time.Time)
				callback(id, job["request"].(string), jobTimestamp.Format("2006-01-02 15:04:05"))
			}
			var retryContext icontext.IContext
			if tmp, ok := ctx.(icontext.IContext); ok {
				retryContext = tmp
			} else {
				retryContext = icontext.New(ctx)
			}
			attempt := job["attempt"].(int)
			_, err := retry(retryContext, job["request"])
			if err != nil && shouldRetry(err, onErrorMessage) && attempt < numberOfRetries {
				attempt := job["attempt"].(int)
				current := job["timestamp"].(time.Time)
				job["attempt"] = attempt + 1
				bg.Queue(schema.Job{
					Id:      nextJobID(id, attempt+1, current),
					Name:    workerName,
					Ctx:     ctx,
					Request: job,
					Delay:   waitTime,
				})
			}
		})
		return func(ctx context.Context, request any) (any, error) {
			var ictx icontext.IContext
			if tmp, ok := ctx.(icontext.IContext); ok {
				ictx = tmp
			} else {
				ictx = icontext.New(ctx)
			}
			response, err := next(ictx, request)
			if err != nil && shouldRetry(err, onErrorMessage) && numberOfRetries > 0 {
				ts := time.Now()
				bg.Queue(schema.Job{
					Id:   nextJobID(id, 1, ts),
					Name: workerName,
					Ctx:  ctx,
					Request: map[string]any{
						"request":   request,
						"attempt":   1,
						"timestamp": ts,
					},
					Delay: waitTime,
				})
			}
			return response, err
		}
	}
}

func shouldRetry(err error, onErrorMessage string) bool {
	return err != nil && err.Error() == onErrorMessage
}

func nextJobID(id string, attempt int, timestamp time.Time) string {
	return fmt.Sprintf("%s:%d:%d", id, attempt, timestamp.UnixNano())
}
