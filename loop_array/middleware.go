package loop_array

import (
	"context"
	"sync"

	"github.com/go-kit/kit/endpoint"
	RUN_WITH_OPTION "github.com/louvri/gokrt/option"
	"github.com/louvri/gosl"
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
				case RUN_WITH_OPTION.RUN_WITH_TRANSACTION:
					opt[RUN_WITH_OPTION.RUN_WITH_TRANSACTION] = true
				default:
					continue
				}
			}

			var kit gosl.Kit
			if opt[RUN_WITH_OPTION.RUN_WITH_TRANSACTION] {
				kit = gosl.New(ctx)
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
					processor := func(ctx context.Context, data interface{}) (interface{}, error) {
						var result interface{}
						var err error
						if arr, ok := data.([]map[string]interface{}); ok {
							for _, item := range arr {
								result, err = e(ctx, preprocessor(item))
								if err != nil && !ignoreError {
									return result, err
								}
							}
						} else if arr, ok := data.([]interface{}); ok {
							for _, item := range arr {
								result, err = e(ctx, preprocessor(item))
								if err != nil && !ignoreError {
									return result, err
								}
							}
						}
						return result, nil
					}
					var err error
					if runAsync {
						var wg sync.WaitGroup
						wg.Add(1)
						go func() {
							if opt[RUN_WITH_OPTION.RUN_WITH_TRANSACTION] {
								if errTransaction := kit.RunInTransaction(ctx, func(ctx context.Context) error {
									_, err := processor(ctx, resp)
									if err != nil {
										return err
									}
									return nil
								}); errTransaction != nil {
									err = errTransaction
								}
							} else {
								_, err = processor(ctx, resp)
							}
							wg.Done()
						}()
						wg.Wait()
						if err != nil && !ignoreError {
							return resp, err
						}

					} else {
						if opt[RUN_WITH_OPTION.RUN_WITH_TRANSACTION] {
							var curr interface{}
							err = kit.RunInTransaction(ctx, func(ctx context.Context) error {
								result, err := processor(ctx, resp)
								curr = result
								return err
							})
							if err != nil && !ignoreError {
								return curr, err
							}
						} else {
							if curr, err := processor(ctx, resp); err != nil && !ignoreError {
								return curr, err
							}
						}
					}
				}
				return resp, nil
			}
		}
	}
}
