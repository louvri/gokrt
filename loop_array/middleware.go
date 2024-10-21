package loop_array

import (
	"context"
	"encoding/json"
	"errors"
	"sync"

	"github.com/go-kit/kit/endpoint"
	RUN_WITH_OPTION "github.com/louvri/gokrt/option"
	"github.com/louvri/gosl"
)

func Middleware(e endpoint.Endpoint, preprocessor func(data interface{}) interface{}, postprocessor func(original, data interface{}, err error), opts ...RUN_WITH_OPTION.Option) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			errorCollection := map[int]interface{}{}

			opt := make(map[RUN_WITH_OPTION.Option]bool)
			for _, option := range opts {
				switch option {
				case RUN_WITH_OPTION.RUN_ASYNC_WAIT:
					opt[RUN_WITH_OPTION.RUN_ASYNC_WAIT] = true
					continue
				case RUN_WITH_OPTION.RUN_WITH_ERROR:
					opt[RUN_WITH_OPTION.RUN_WITH_ERROR] = true
					continue
				case RUN_WITH_OPTION.RUN_IN_TRANSACTION:
					opt[RUN_WITH_OPTION.RUN_IN_TRANSACTION] = true
				default:
					continue
				}
			}
			var kit gosl.Kit
			if opt[RUN_WITH_OPTION.RUN_IN_TRANSACTION] {
				kit = gosl.New(ctx)
			}
			ori, err := next(ctx, req)
			if err != nil {
				return nil, err
			} else if ori != nil {
				run := func(data interface{}, index int) (interface{}, error) {
					inner := func(index int) (interface{}, error) {
						var req interface{}
						if preprocessor != nil {
							req = preprocessor(data)
						} else {
							req = data
						}
						resp, err := e(ctx, req)
						if err != nil {
							if !opt[RUN_WITH_OPTION.RUN_WITH_ERROR] {
								return nil, err
							}
							errorCollection[index] = err.Error()

						}
						if postprocessor != nil {
							postprocessor(data, resp, err)
						}
						return resp, nil
					}
					if opt[RUN_WITH_OPTION.RUN_ASYNC_WAIT] {
						var response interface{}
						var err error
						var wg sync.WaitGroup
						wg.Add(1)
						go func() {
							response, err = inner(index)
							wg.Done()
						}()
						wg.Wait()
						return response, err
					}
					return inner(index)
				}
				if opt[RUN_WITH_OPTION.RUN_IN_TRANSACTION] {
					if err := kit.RunInTransaction(ctx, func(ctx context.Context) error {
						if arr, ok := ori.([]map[string]interface{}); ok {
							for index, item := range arr {
								_, err := run(item, index)
								if err != nil {
									return err
								}
							}
							return nil
						} else if arr, ok := ori.([]interface{}); ok {
							for index, item := range arr {
								_, err := run(item, index)
								if err != nil {
									return err
								}
							}
						}
						return nil
					}); err != nil {
						return nil, err
					}
				} else {
					if arr, ok := ori.([]map[string]interface{}); ok {
						for index, item := range arr {
							_, err := run(item, index)
							if err != nil {
								return nil, err
							}
						}
					} else if arr, ok := ori.([]interface{}); ok {
						for index, item := range arr {
							_, err := run(item, index)
							if err != nil {
								return nil, err
							}
						}
					}
				}
			}
			var errorOutcome error
			if len(errorCollection) > 0 {
				marshalled, _ := json.Marshal(errorCollection)
				errorOutcome = errors.New(string(marshalled))
			}
			return ori, errorOutcome
		}
	}
}
