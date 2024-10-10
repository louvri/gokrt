package loop_next

import (
	"context"
	"sync"
	"time"

	"github.com/go-kit/kit/endpoint"

	RUN_WITH_OPTION "github.com/louvri/gokrt/option"
	"github.com/louvri/gokrt/sys_key"
	"github.com/louvri/gosl"
)

// loop
func Middleware(
	comparator func(prev, curr interface{}) bool,
	modifier func(req interface{}, next interface{}) interface{},
	postprocessor func(original, data interface{}, err error),
	opts ...RUN_WITH_OPTION.Option) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
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
			errors := make([]string, 0)
			if opt[RUN_WITH_OPTION.RUN_IN_TRANSACTION] {
				kit = gosl.New(ctx)
			}

			var prev, curr interface{}
			var err error
			var response interface{}
			curr = make([]map[string]interface{}, 0)
			prevRequest := req

			run := func(ctx context.Context) (interface{}, error) {
				inner := func() (interface{}, error) {
					currReq := modifier(prevRequest, curr)
					prev = curr
					ctx = context.WithValue(ctx, sys_key.DATA_REF, prev)
					curr, err = next(ctx, currReq)
					response = curr
					if err != nil {
						ctx = context.WithValue(ctx, sys_key.EOF, "err")
						errors = append(errors, err.Error())
						response, err = next(ctx, nil)
						if !opt[RUN_WITH_OPTION.RUN_WITH_ERROR] {
							return nil, err
						}
					}

					if postprocessor != nil {
						postprocessor(req, curr, err)
					}

					ctx = context.WithValue(ctx, sys_key.SOF, false)
					time.Sleep(0)
					return response, nil
				}

				if opt[RUN_WITH_OPTION.RUN_ASYNC_WAIT] {
					var response interface{}
					var err error
					var wg sync.WaitGroup
					wg.Add(1)
					go func() {
						response, err = inner()
						wg.Done()
					}()
					wg.Wait()
					return response, err
				}
				return inner()
			}

			ctx = context.WithValue(ctx, sys_key.SOF, true)
			eof := ctx.Value(sys_key.EOF)
			if eof != nil {
				return next(ctx, nil)
			}

			if opt[RUN_WITH_OPTION.RUN_IN_TRANSACTION] {
				if err := kit.RunInTransaction(ctx, func(ctx context.Context) error {
					for !comparator(prev, curr) {
						response, err = run(ctx)
						if err != nil {
							return err
						}
					}
					ctx = context.WithValue(ctx, sys_key.EOF, "eof")
					if eofResponse, eofErr := next(ctx, nil); eofErr != nil {
						return eofErr
					} else {
						response = eofResponse
						return nil
					}

				}); err != nil {
					return nil, err
				}
				return response, nil
			}

			for !comparator(prev, curr) {
				response, err = run(ctx)
				if err != nil {
					return nil, err
				}
			}
			ctx = context.WithValue(ctx, sys_key.EOF, "eof")
			if eofResponse, eofErr := next(ctx, nil); eofErr != nil {
				return nil, eofErr
			} else {
				response = eofResponse
			}
			return response, err
		}
	}
}
