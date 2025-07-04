package alter

import (
	"context"
	"sync"

	"github.com/go-kit/kit/endpoint"
	RUN_WITH_OPTION "github.com/louvri/gokrt/option"
)

func Middleware(
	e endpoint.Endpoint,
	preprocessor func(data interface{}, err error) interface{},
	postprocessor func(original interface{}, data interface{}, err error) (interface{}, error),
	opts ...RUN_WITH_OPTION.Option) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			opt := make(map[RUN_WITH_OPTION.Option]bool)
			for _, option := range opts {
				opt[option] = true
			}
			var original interface{}
			var err error
			runOnError := opt[RUN_WITH_OPTION.RUN_WITH_ERROR]
			original, err = next(ctx, req)
			if err != nil && !runOnError {
				return original, err
			}

			if original != nil {
				result := preprocessor(original, err)
				if result != nil {
					var altered interface{}
					if runAsync := opt[RUN_WITH_OPTION.RUN_ASYNC_WAIT]; runAsync {
						var wg sync.WaitGroup
						wg.Add(1)
						go func() {
							defer wg.Done()
							altered, err = e(ctx, req)
						}()
						wg.Wait()
						return postprocessor(original, altered, err)
					} else {
						altered, err = e(ctx, result)
						return postprocessor(original, altered, err)
					}

				} else {
					return nil, nil
				}
			}
			return original, err

		}
	}
}
