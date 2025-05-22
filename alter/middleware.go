package alter

import (
	"context"
	"sync"

	"github.com/go-kit/kit/endpoint"
	RUN_WITH_OPTION "github.com/louvri/gokrt/option"
	"github.com/louvri/gokrt/wrapper"
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
			original, err = next(ctx, req)

			runOnError := opt[RUN_WITH_OPTION.RUN_WITH_ERROR]
			if runOnError || err == nil && original != nil {
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
						var extracted any
						if tmp, ok := altered.(wrapper.Wrapper); ok {
							extracted = tmp
						} else {
							extracted = altered
						}
						return postprocessor(original, extracted, err)
					} else {
						altered, err = e(ctx, result)
						var extracted any
						if tmp, ok := altered.(wrapper.Wrapper); ok {
							extracted = tmp
						} else {
							extracted = altered
						}
						return postprocessor(original, extracted, err)
					}

				} else {
					return nil, nil
				}
			}
			return original, err

		}
	}
}
