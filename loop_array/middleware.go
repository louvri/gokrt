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

func Middleware(e endpoint.Endpoint, preprocessor func(data any, err error) any, postprocessor func(original, data any, err error), opts ...RUN_WITH_OPTION.Option) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req any) (any, error) {
			errorCollection := make([]map[string]any, 0)

			opt := make(map[RUN_WITH_OPTION.Option]bool)
			for _, option := range opts {
				opt[option] = true
			}
			ori, err := next(ctx, req)
			if err != nil {
				return nil, err
			} else if ori != nil {
				run := func(data any, index int) (any, error) {
					inner := func(index int) (any, error) {
						var req any
						if preprocessor != nil {
							req = preprocessor(data, err)
						} else {
							req = data
						}
						resp, err := e(ctx, req)
						if err != nil {
							if !opt[RUN_WITH_OPTION.RUN_WITH_ERROR] {
								return nil, err
							} else {
								errorCollection = append(errorCollection, map[string]any{
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
					return inner(index)
				}

				handler := func(data any) error {
					if tobeHandle, ok := data.([]map[string]any); ok {
						for index, item := range tobeHandle {
							_, err := run(item, index)
							if err != nil {
								return err
							}
						}
					} else if tobeHandle, ok := data.([]any); ok {
						for index, item := range tobeHandle {
							_, err := run(item, index)
							if err != nil {
								return err
							}
						}
					}
					return nil
				}

				if opt[RUN_WITH_OPTION.RUN_IN_TRANSACTION] {
					ctx, kit := gosl.New(ctx)
					if opt[RUN_WITH_OPTION.RUN_ASYNC_WAIT] {
						var err error
						var wg sync.WaitGroup
						wg.Add(1)
						go func() error {
							defer wg.Done()
							if err := kit.RunInTransaction(ctx, func(ctx context.Context) error {
								err = handler(ori)
								if err != nil {
									return err
								}
								return nil
							}); err != nil {
								return err
							}
							return nil
						}()
						wg.Wait()
						if err != nil {
							return nil, err
						}
					} else {
						var err error
						go func() error {
							if err := kit.RunInTransaction(ctx, func(ctx context.Context) error {
								err = handler(ori)
								if err != nil {
									return err
								}
								return nil
							}); err != nil {
								return err
							}
							return nil
						}()
						if err != nil {
							return nil, err
						}
					}

				} else {
					if opt[RUN_WITH_OPTION.RUN_ASYNC_WAIT] {
						var err error
						var wg sync.WaitGroup
						wg.Add(1)
						go func() error {
							defer wg.Done()
							err = handler(ori)
							if err != nil {
								return err
							}
							return nil
						}()
						wg.Wait()
						if err != nil {
							return nil, err
						}
					} else {
						var err error
						go func() error {
							err = handler(ori)
							if err != nil {
								return err
							}
							return nil
						}()
						if err != nil {
							return nil, err
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
