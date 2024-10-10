package after

import (
	"context"
	"sync"

	"github.com/go-kit/kit/endpoint"
	icontext "github.com/louvri/gokrt/icontext"
	RUN_WITH_OPTION "github.com/louvri/gokrt/option"
)

func Middleware(
	e endpoint.Endpoint,
	preprocessor func(data interface{}, err error) interface{},
	postprocessor func(data interface{}, err error),
	opts ...RUN_WITH_OPTION.Option) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			opt := make(map[RUN_WITH_OPTION.Option]bool)
			for _, option := range opts {
				switch option {
				case RUN_WITH_OPTION.RUN_ASYNC_WAIT:
					opt[RUN_WITH_OPTION.RUN_ASYNC_WAIT] = true
				case RUN_WITH_OPTION.RUN_WITH_ERROR:
					opt[RUN_WITH_OPTION.RUN_WITH_ERROR] = true
				}
			}
			resp, err := next(ctx, req)
			runOnError := opt[RUN_WITH_OPTION.RUN_WITH_ERROR]
			if runOnError || err == nil {
				result := preprocessor(resp, err)
				if result != nil {
					if _, ok := ctx.(*icontext.CopyContext); !ok {
						ctx = icontext.New(ctx)
					}
					if runAsync := opt[RUN_WITH_OPTION.RUN_ASYNC_WAIT]; runAsync {
						var wg sync.WaitGroup
						wg.Add(1)
						go func() {
							e(ctx, result)
							if postprocessor != nil {
								postprocessor(resp, err)
							}
							wg.Done()
						}()
						wg.Wait()
					} else {
						go func() {
							e(ctx, result)
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
