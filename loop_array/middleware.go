package loop_array

import (
	"context"
	"errors"
	"strings"
	"sync"

	"github.com/go-kit/kit/endpoint"
	RUN_WITH_OPTION "github.com/louvri/gokrt/option"
	"github.com/louvri/gosl"
)

func Middleware(e endpoint.Endpoint, preprocessor func(data interface{}) interface{}, postprocessor func(original, data interface{}, err error), opts ...RUN_WITH_OPTION.Option) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			opt := make(map[RUN_WITH_OPTION.Option]bool)
			var IGNORE_ERROR, RUN_ASYNCHRONOUSLY bool
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
				case RUN_WITH_OPTION.RUN_IN_TRANSACTION:
					opt[RUN_WITH_OPTION.RUN_IN_TRANSACTION] = true
				default:
					continue
				}
			}

			var kit gosl.Kit
			BUNDLE_OF_ERRORS := make([]string, 0)
			var RUN_IN_TRANSACTION bool
			if opt[RUN_WITH_OPTION.RUN_IN_TRANSACTION] {
				RUN_IN_TRANSACTION = true
				kit = gosl.New(ctx)
			}

			if tmp, ok := opt[RUN_WITH_OPTION.RUN_ASYNC]; ok && RUN_ASYNCHRONOUSLY {
				RUN_ASYNCHRONOUSLY = tmp
			}

			if tmp, ok := opt[RUN_WITH_OPTION.RUN_WITH_ERROR]; ok && tmp {
				IGNORE_ERROR = tmp
			}

			resp, err := next(ctx, req)
			if err != nil && !IGNORE_ERROR {
				return resp, err
			} else {
				if resp != nil {
					processor := func(ctx context.Context, data interface{}) (interface{}, error) {
						var result interface{}
						var err error
						if arr, ok := data.([]map[string]interface{}); ok {
							for _, item := range arr {
								result, err = e(ctx, preprocessor(item))
								if postprocessor != nil {
									postprocessor(preprocessor(item), result, err)
								}
								if err != nil && RUN_IN_TRANSACTION {
									return result, err
								} else if err != nil {
									BUNDLE_OF_ERRORS = append(BUNDLE_OF_ERRORS, err.Error())
								}
							}
						} else if arr, ok := data.([]interface{}); ok {
							for _, item := range arr {
								result, err = e(ctx, preprocessor(item))
								if postprocessor != nil {
									postprocessor(preprocessor(item), result, err)
								}
								if err != nil && RUN_IN_TRANSACTION {
									return result, err
								} else if err != nil {
									BUNDLE_OF_ERRORS = append(BUNDLE_OF_ERRORS, err.Error())
								}
							}
						}

						if len(BUNDLE_OF_ERRORS) > 0 && !IGNORE_ERROR {
							return result, errors.New(strings.Join(BUNDLE_OF_ERRORS, " || "))
						} else {
							return result, nil
						}
					}
					var err error
					if RUN_ASYNCHRONOUSLY {
						var wg sync.WaitGroup
						wg.Add(1)
						go func() {
							if opt[RUN_WITH_OPTION.RUN_IN_TRANSACTION] {
								if ERROR_IN_TRANSACTION := kit.RunInTransaction(ctx, func(ctx context.Context) error {
									_, err := processor(ctx, resp)
									if err != nil {
										return err
									}
									return nil
								}); ERROR_IN_TRANSACTION != nil {
									BUNDLE_OF_ERRORS = append(BUNDLE_OF_ERRORS, ERROR_IN_TRANSACTION.Error())
								}
							} else {
								_, err = processor(ctx, resp)
								if err != nil {
									BUNDLE_OF_ERRORS = append(BUNDLE_OF_ERRORS, err.Error())
								}
							}
							wg.Done()
						}()
						wg.Wait()
						if len(BUNDLE_OF_ERRORS) > 0 && !IGNORE_ERROR {
							return resp, errors.New(strings.Join(BUNDLE_OF_ERRORS, " || "))
						}

					} else {
						var curr interface{}
						if opt[RUN_WITH_OPTION.RUN_IN_TRANSACTION] {
							err = kit.RunInTransaction(ctx, func(ctx context.Context) error {
								result, err := processor(ctx, resp)
								curr = result
								return err
							})
							if err != nil {
								BUNDLE_OF_ERRORS = append(BUNDLE_OF_ERRORS, err.Error())
							}
						} else {
							curr, err = processor(ctx, resp)
						}
						resp = curr
					}
				}
				if len(BUNDLE_OF_ERRORS) > 0 && !IGNORE_ERROR {
					return resp, errors.New(strings.Join(BUNDLE_OF_ERRORS, " || "))
				}
				return resp, nil
			}
		}
	}
}
