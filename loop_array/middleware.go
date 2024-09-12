package loop_array

import (
	"context"
	"sync"

	"github.com/go-kit/kit/endpoint"
	RUN_WITH_OPTION "github.com/louvri/gokrt/option"
)

func Middleware(e endpoint.Endpoint, preprocessor func(data interface{}) interface{}, opts ...RUN_WITH_OPTION.Option) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			opt := make(map[RUN_WITH_OPTION.Option]bool)
			var ignoreError, runAsync bool
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

			if tmp, ok := opt[RUN_WITH_OPTION.RUN_ASYNC]; ok && runAsync {
				runAsync = tmp
			}

			if tmp, ok := opt[RUN_WITH_OPTION.RUN_WITH_ERROR]; ok && tmp {
				ignoreError = tmp
			}

			resp, err := next(ctx, req)
			if err != nil && !ignoreError {
				return resp, err
			} else {
				if resp != nil {
					var err error
					if runAsync {
						var wg sync.WaitGroup
						wg.Add(1)
						go func() {
							if arr, ok := resp.([]map[string]interface{}); ok {
								for _, item := range arr {
									if _, tmpErr := e(ctx, preprocessor(item)); tmpErr != nil {
										err = tmpErr
									}

								}
							} else if arr, ok := resp.([]interface{}); ok {
								for _, item := range arr {
									if _, tmpErr := e(ctx, preprocessor(item)); tmpErr != nil {
										err = tmpErr
									}
								}
							}
							wg.Done()
						}()
						wg.Wait()
						if err != nil && !ignoreError {
							return resp, err
						}

					} else {
						if arr, ok := resp.([]map[string]interface{}); ok {
							for _, item := range arr {
								curr, err := e(ctx, preprocessor(item))
								if err != nil && !ignoreError {
									return curr, err
								}
							}
						} else if arr, ok := resp.([]interface{}); ok {
							for _, item := range arr {
								curr, err := e(ctx, preprocessor(item))
								if err != nil && !ignoreError {
									return curr, err
								}
							}
						}
					}
				}
				return resp, nil
			}
		}
	}
}
