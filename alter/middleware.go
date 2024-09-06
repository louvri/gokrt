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
				switch option {
				case RUN_WITH_OPTION.RUN_ASYNC:
					opt[RUN_WITH_OPTION.RUN_ASYNC] = true
					continue
				case RUN_WITH_OPTION.RUN_WITH_ERROR:
					opt[RUN_WITH_OPTION.RUN_WITH_ERROR] = true
					continue
				case RUN_WITH_OPTION.EXECUTE_AFTER:
					opt[RUN_WITH_OPTION.EXECUTE_AFTER] = true
				case RUN_WITH_OPTION.EXECUTE_BEFORE:
					opt[RUN_WITH_OPTION.EXECUTE_BEFORE] = true
				default:
					continue
				}
			}
			var original interface{}
			var err error
			if opt[RUN_WITH_OPTION.EXECUTE_AFTER] {
				original, err = next(ctx, req)
			} else if opt[RUN_WITH_OPTION.EXECUTE_BEFORE] {
				original, err = e(ctx, req)
			}
			runOnError := opt[RUN_WITH_OPTION.RUN_WITH_ERROR]
			if original != nil && (err == nil || runOnError) {
				result := preprocessor(original, err)
				if result != nil {
					var altered interface{}
					if runAsync := opt[RUN_WITH_OPTION.RUN_ASYNC]; runAsync {
						var wg sync.WaitGroup
						wg.Add(1)
						go func() {
							defer wg.Done()
							if opt[RUN_WITH_OPTION.EXECUTE_AFTER] {
								altered, err = e(ctx, req)
							} else if opt[RUN_WITH_OPTION.EXECUTE_BEFORE] {
								altered, err = next(ctx, req)
							}
						}()
						wg.Wait()
						return postprocessor(original, altered, err)
					} else {

						if opt[RUN_WITH_OPTION.EXECUTE_AFTER] {
							altered, err = e(ctx, result)
						} else if opt[RUN_WITH_OPTION.EXECUTE_BEFORE] {
							original, err = next(ctx, req)
						}
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
