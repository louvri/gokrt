package before

import (
	"context"
	"sync"

	"github.com/go-kit/kit/endpoint"
	"github.com/louvri/gokrt/icontext"
	RUN_WITH_OPTION "github.com/louvri/gokrt/option"
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
				switch option {
				case RUN_WITH_OPTION.RUN_ASYNC_WAIT:
					opt[RUN_WITH_OPTION.RUN_ASYNC_WAIT] = true
				}
			}
			var result interface{}
			if preprocessor != nil {
				result = preprocessor(req)
			} else {
				result = req
			}
			if result != nil {
				if _, ok := ctx.(*icontext.CopyContext); !ok {
					ctx = icontext.New(ctx)
				}
				if runAsync := opt[RUN_WITH_OPTION.RUN_ASYNC_WAIT]; runAsync {
					var wg sync.WaitGroup
					wg.Add(1)
					go func() {
						postdata, err := e(ctx, result)
						if postprocessor != nil {
							postprocessor(postdata, err)
						}
						wg.Done()
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
