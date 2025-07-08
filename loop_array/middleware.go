package loop_array

import (
	"context"
	"encoding/json"
	"errors"
	"sync"

	"github.com/go-kit/kit/endpoint"
	icontext "github.com/louvri/gokrt/context"
	RUN_WITH_OPTION "github.com/louvri/gokrt/option"
	"github.com/louvri/gokrt/sys_key"
	"github.com/louvri/gosl"
)

func Middleware(e endpoint.Endpoint, preprocessor func(data interface{}) interface{}, postprocessor func(original, data interface{}, err error), opts ...RUN_WITH_OPTION.Option) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			errorCollection := make([]map[string]interface{}, 0)

			opt := make(map[RUN_WITH_OPTION.Option]bool)
			for _, option := range opts {
				opt[option] = true
			}
			/*var kit gosl.Kit
			if opt[RUN_WITH_OPTION.RUN_IN_TRANSACTION] {
				kit = gosl.New(ctx)
			}*/
			if _, ok := ctx.Value(sys_key.INTERNAL_CONTEXT).(*icontext.Context); !ok {
				ctx = icontext.New(ctx)
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
							} else {
								errorCollection = append(errorCollection, map[string]interface{}{
									"lineNumber": index,
									"error":      err.Error(),
								})
							}
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
							defer wg.Done()
							response, err = inner(index)
						}()
						wg.Wait()
						return response, err
					}
					return inner(index)
				}
				if opt[RUN_WITH_OPTION.RUN_IN_TRANSACTION] {
					ctx, kit := gosl.New(ctx)
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
