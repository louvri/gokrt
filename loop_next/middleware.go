package loop_next

import (
	"context"
	"sync"
	"time"

	"github.com/go-kit/kit/endpoint"
	"github.com/louvri/gokrt/sys_key"
	"github.com/louvri/gosl"

	RUN_WITH_OPTION "github.com/louvri/gokrt/option"
)

// loop
func Middleware(
	comparator func(prev, curr interface{}) bool,
	modifier func(req interface{}, next interface{}) interface{},
	opts ...RUN_WITH_OPTION.Option) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			opt := make(map[RUN_WITH_OPTION.Option]bool)
			// bundledErr := make([]interface{}, 0)
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
			eof := ctx.Value(sys_key.EOF)
			if eof != nil {
				return next(ctx, nil)
			} else {
				process := func(ctx context.Context) (interface{}, error) {
					var prev, curr interface{}
					var err error
					var response interface{}
					curr = make([]map[string]interface{}, 0)
					modifiedReq := req
					ctx = context.WithValue(ctx, sys_key.SOF, true)
					for !comparator(prev, curr) {
						modifiedReq := modifier(modifiedReq, curr)
						prev = curr
						ctx = context.WithValue(ctx, sys_key.DATA_REF, prev)
						curr, err = next(ctx, modifiedReq)
						if !opt[RUN_WITH_OPTION.RUN_WITH_ERROR] && err != nil {
							ctx = context.WithValue(ctx, sys_key.EOF, "err")
							response, err = next(ctx, nil)
						}
						ctx = context.WithValue(ctx, sys_key.SOF, false)
						time.Sleep(0)
					}
					ctx = context.WithValue(ctx, sys_key.EOF, "eof")
					if eofResponse, eofErr := next(ctx, nil); eofErr != nil {
						err = eofErr
					} else {
						response = eofResponse
					}
					return response, err
				}
				if opt[RUN_WITH_OPTION.RUN_ASYNC] {
					var response interface{}
					var err error
					var wg sync.WaitGroup
					wg.Add(1)
					go func() {
						if opt[RUN_WITH_OPTION.RUN_WITH_TRANSACTION] {
							if errTransaction := kit.RunInTransaction(ctx, func(ctx context.Context) error {
								resp, err := process(ctx)
								response = resp
								if err != nil {
									return err
								}
								return nil
							}); errTransaction != nil {
								err = errTransaction
							}
						} else {
							response, err = process(ctx)
						}
						wg.Done()
					}()
					wg.Wait()
					return response, err
				} else {
					var response interface{}
					var err error
					if opt[RUN_WITH_OPTION.RUN_WITH_TRANSACTION] {
						if errTransaction := kit.RunInTransaction(ctx, func(ctx context.Context) error {
							response, err = process(ctx)
							if err != nil {
								return err
							}
							return nil
						}); errTransaction != nil {
							err = errTransaction
						}
					} else {
						response, err = process(ctx)
					}
					return response, err
				}
			}
		}
	}
}
