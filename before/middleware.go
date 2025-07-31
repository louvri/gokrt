package before

import (
	"context"
	"sync"

	"github.com/go-kit/kit/endpoint"
	icontext "github.com/louvri/gokrt/context"
	RUN_WITH_OPTION "github.com/louvri/gokrt/option"
	"github.com/louvri/gokrt/sys_key"
)

func Middleware(
	e endpoint.Endpoint,
	preprocessor func(data interface{}) interface{},
	postprocessor func(data interface{}, err error),
	opts ...RUN_WITH_OPTION.Option) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			opt := make(map[RUN_WITH_OPTION.Option]bool)
			for _, option := range opts {
				opt[option] = true
			}

			if _, ok := ctx.Value(sys_key.INTERNAL_CONTEXT).(*icontext.Context); !ok {
				ctx = icontext.New(ctx)
			}
			var result interface{}
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
						postdata, err := e(ctx, result)
						if postprocessor != nil {
							postprocessor(postdata, err)
						}
					}()
					wg.Wait()
				} else {
					go func() {
						postdata, err := e(ctx, result)
						if postprocessor != nil {
							postprocessor(postdata, err)
						}
					}()
				}
			}
			return next(ctx, req)
		}
	}
}
