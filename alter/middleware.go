package alter

import (
	"context"
	"sync"

	"github.com/go-kit/kit/endpoint"
)

type Option string

var RUN_WITH_ERROR Option = "RUN ON ERROR"
var RUN_ASYNC Option = "RUN ASYNC"

func Middleware(
	e endpoint.Endpoint,
	preprocessor func(data interface{}, err error) interface{},
	postprocessor func(original interface{}, data interface{}, err error) (interface{}, error),
	opts ...Option) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			opt := make(map[Option]bool)
			for _, option := range opts {
				switch option {
				case RUN_ASYNC:
					opt[RUN_ASYNC] = true
					continue
				case RUN_WITH_ERROR:
					opt[RUN_WITH_ERROR] = true
					continue
				default:
					continue
				}
			}
			original, err := next(ctx, req)
			runOnError := opt[RUN_WITH_ERROR]
			if original != nil && (err == nil || runOnError) {
				result := preprocessor(original, err)
				if result != nil {
					if runAsync := opt[RUN_ASYNC]; runAsync {
						var wg sync.WaitGroup
						var altered interface{}
						wg.Add(1)
						go func() {
							defer wg.Done()
							altered, err = e(ctx, result)
						}()
						wg.Wait()
						return postprocessor(original, altered, err)
					} else {
						altered, err := e(ctx, result)
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
