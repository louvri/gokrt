package after

import (
	"context"
	"sync"

	"github.com/go-kit/kit/endpoint"
	icontext "github.com/louvri/gokrt/context"
	RUN_WITH_OPTION "github.com/louvri/gokrt/option"
)

func Middleware(
	e endpoint.Endpoint,
	preprocessor func(data any, err error) any,
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
			resp, err := next(ictx, req)
			runOnError := opt[RUN_WITH_OPTION.RUN_WITH_ERROR]
			if runOnError || err == nil {
				result := resp
				if preprocessor != nil {
					result = preprocessor(resp, err)
				}
				if result != nil {

					if runAsync := opt[RUN_WITH_OPTION.RUN_ASYNC_WAIT]; runAsync {
						var wg sync.WaitGroup
						wg.Add(1)
						go func() {
							defer wg.Done()
							e(ictx.WithoutDeadline(), result)
							if postprocessor != nil {
								postprocessor(resp, err)
							}
						}()
						wg.Wait()
					} else {
						go func() {
							e(ictx.WithoutDeadline(), result)
							if postprocessor != nil {
								postprocessor(resp, err)
							}
						}()
					}
				}
			}

			return resp, err
		}
	}
}
