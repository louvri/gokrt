package before

import (
	"context"
	"sync"

	"github.com/go-kit/kit/endpoint"
	icontext "github.com/louvri/gokrt/context"
	RUN_WITH_OPTION "github.com/louvri/gokrt/option"
)

func Middleware(
	e endpoint.Endpoint,
	preprocessor func(data any) any,
	postprocessor func(data any, err error),
	opts ...RUN_WITH_OPTION.Option) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req any) (any, error) {
			var ictx icontext.IContext
			if tmp, ok := ctx.(icontext.IContext); ok {
				ictx = tmp
			} else {
				ictx = icontext.New(ctx)
			}
			opt := make(map[RUN_WITH_OPTION.Option]bool)
			for _, option := range opts {
				opt[option] = true
			}
			var result any
			if preprocessor != nil {
				result = preprocessor(req)
			} else {
				result = req
			}
			if result != nil {
				if runAsync := opt[RUN_WITH_OPTION.RUN_ASYNC_WAIT]; runAsync {
					var wg sync.WaitGroup
					wg.Add(1)
					go func() {
						defer wg.Done()
						postdata, err := e(ictx.WithoutDeadline(ictx), result)
						if postprocessor != nil {
							postprocessor(postdata, err)
						}
					}()
					wg.Wait()
				} else {
					go func() {
						postdata, err := e(ictx.WithoutDeadline(ictx), result)
						if postprocessor != nil {
							postprocessor(postdata, err)
						}
					}()
				}
			}
			return next(ictx, req)
		}
	}
}
