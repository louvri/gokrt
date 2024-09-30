package loop_next

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/go-kit/kit/endpoint"
	"github.com/louvri/gokrt/sys_key"
	"github.com/louvri/gosl"

	RUN_OPTION "github.com/louvri/gokrt/option"
)

// loop
func Middleware(
	comparator func(prev, curr interface{}) bool,
	modifier func(req interface{}, next interface{}) interface{},
	postprocessor func(original, data interface{}, err error),
	opts ...RUN_OPTION.Option) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			opt := make(map[RUN_OPTION.Option]bool)
			BUNDLE_OF_ERRORS := make([]string, 0)
			for _, option := range opts {
				switch option {
				case RUN_OPTION.RUN_ASYNC:
					opt[RUN_OPTION.RUN_ASYNC] = true
					continue
				case RUN_OPTION.RUN_WITH_ERROR:
					opt[RUN_OPTION.RUN_WITH_ERROR] = true
					continue
				case RUN_OPTION.EXECUTE_AFTER:
					opt[RUN_OPTION.EXECUTE_AFTER] = true
				case RUN_OPTION.EXECUTE_BEFORE:
					opt[RUN_OPTION.EXECUTE_BEFORE] = true
				case RUN_OPTION.RUN_IN_TRANSACTION:
					opt[RUN_OPTION.RUN_IN_TRANSACTION] = true
				default:
					continue
				}
			}
			var kit gosl.Kit
			if opt[RUN_OPTION.RUN_IN_TRANSACTION] {
				kit = gosl.New(ctx)
			}
			eof := ctx.Value(sys_key.EOF)
			if eof != nil {
				return next(ctx, nil)
			} else {
				// anonym function for data processing
				processor := func(ctx context.Context) (interface{}, error) {
					var prev, curr interface{}
					var err error
					var response interface{}
					curr = make([]map[string]interface{}, 0)
					prevRequest := req
					ctx = context.WithValue(ctx, sys_key.SOF, true)
					for !comparator(prev, curr) {
						currReq := modifier(prevRequest, curr)
						prev = curr
						ctx = context.WithValue(ctx, sys_key.DATA_REF, prev)
						curr, err = next(ctx, currReq)
						if postprocessor != nil {
							postprocessor(req, curr, err)
						}
						if opt[RUN_OPTION.RUN_IN_TRANSACTION] && err != nil {
							return nil, err
						} else if err != nil {
							ctx = context.WithValue(ctx, sys_key.EOF, "err")
							BUNDLE_OF_ERRORS = append(BUNDLE_OF_ERRORS, err.Error())
							response, err = next(ctx, nil)
							// if running in transaction, it will break the loop once the errors appear
							if err != nil && opt[RUN_OPTION.RUN_IN_TRANSACTION] {
								return response, err
							} else if err != nil {
								BUNDLE_OF_ERRORS = append(BUNDLE_OF_ERRORS, err.Error())
							}
						}
						ctx = context.WithValue(ctx, sys_key.SOF, false)
						time.Sleep(0)
					}
					ctx = context.WithValue(ctx, sys_key.EOF, "eof")
					if eofResponse, eofErr := next(ctx, nil); eofErr != nil {
						BUNDLE_OF_ERRORS = append(BUNDLE_OF_ERRORS, eofErr.Error())
					} else {
						response = eofResponse
					}
					if len(BUNDLE_OF_ERRORS) > 0 {
						return response, errors.New(strings.Join(BUNDLE_OF_ERRORS, " || "))
					} else {
						return response, nil
					}

				}

				if opt[RUN_OPTION.RUN_ASYNC] {
					var response interface{}
					var err error
					var wg sync.WaitGroup
					wg.Add(1)
					go func() {
						if opt[RUN_OPTION.RUN_IN_TRANSACTION] {
							if errTransaction := kit.RunInTransaction(ctx, func(ctx context.Context) error {
								resp, err := processor(ctx)
								response = resp
								if err != nil {
									return err
								}
								return nil
							}); errTransaction != nil {
								err = errTransaction
							}
						} else {
							response, err = processor(ctx)
						}
						wg.Done()
					}()
					wg.Wait()
					return response, err
				} else {
					var response interface{}
					var err error
					if opt[RUN_OPTION.RUN_IN_TRANSACTION] {
						if ERROR_ON_TRANSACTION := kit.RunInTransaction(ctx, func(ctx context.Context) error {
							response, err = processor(ctx)
							if err != nil {
								return err
							}
							return nil
						}); ERROR_ON_TRANSACTION != nil {
							BUNDLE_OF_ERRORS = append(BUNDLE_OF_ERRORS, ERROR_ON_TRANSACTION.Error())
						}
					} else {
						response, err = processor(ctx)
					}
					if opt[RUN_OPTION.RUN_WITH_ERROR] && err != nil {
						return response, err
					}
					return response, nil
				}
			}
		}
	}
}
