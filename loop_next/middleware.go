package loop_next

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/go-kit/kit/endpoint"

	icontext "github.com/louvri/gokrt/context"
	RUN_WITH_OPTION "github.com/louvri/gokrt/option"
	"github.com/louvri/gokrt/sys_key"
	"github.com/louvri/gosl"
)

// loop
func Middleware(
	comparator func(prev, curr any) bool,
	modifier func(req any, next any) any,
	postprocessor func(original, data any, err error),
	opts ...RUN_WITH_OPTION.Option) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req any) (any, error) {
			var ok bool
			var ictx *icontext.Context
			if ictx, ok = ctx.Value(sys_key.GOKRT_CONTEXT).(*icontext.Context); !ok {
				ictx = icontext.New(ctx).(*icontext.Context)
			}
			opt := make(map[RUN_WITH_OPTION.Option]bool)
			for _, option := range opts {
				opt[option] = true
			}
			var prev, curr any
			var err error
			var response any
			curr = make([]map[string]any, 0)
			errorCollection := make([]map[string]any, 0)
			run := func(iteration int, ctx context.Context) (any, error) {
				inner := func(iteration int) (any, error) {
					prev = curr
					currReq := modifier(req, curr)
					response, err = next(ctx, currReq)
					curr = response
					if err != nil {
						if !opt[RUN_WITH_OPTION.RUN_WITHOUT_FILE_DESCRIPTOR] {
							// ctx = context.WithValue(ctx, sys_key.EOF, "err")
							ictx.Set(sys_key.EOF, "err")
							response, _ = next(ctx, nil)
						}
						if !opt[RUN_WITH_OPTION.RUN_WITH_ERROR] {
							return nil, err
						} else {
							errorCollection = append(errorCollection, map[string]any{
								"iteration": iteration,
								"error":     err.Error(),
							})
						}
					}
					if postprocessor != nil {
						postprocessor(req, curr, err)
					}

					time.Sleep(0)
					return response, nil
				}

				if opt[RUN_WITH_OPTION.RUN_ASYNC_WAIT] {
					var response any
					var err error
					var wg sync.WaitGroup
					wg.Add(1)
					go func() {
						defer wg.Done()
						response, err = inner(iteration)
					}()
					wg.Wait()
					return response, err
				}
				return inner(iteration)
			}

			// eof := ctx.Value(sys_key.EOF)
			eof := ictx.Get(sys_key.EOF)
			if eof != nil {
				return next(ctx, nil)
			}
			var idx int
			if !opt[RUN_WITH_OPTION.RUN_WITHOUT_FILE_DESCRIPTOR] {
				// ctx = context.WithValue(ctx, sys_key.SOF, true)
				ictx.Set(sys_key.SOF, true)
			}
			if opt[RUN_WITH_OPTION.RUN_IN_TRANSACTION] {
				ctx, kit := gosl.New(ictx)
				if err := kit.RunInTransaction(ctx, func(ctx context.Context) error {
					for !comparator(prev, curr) {
						response, err = run(idx, ctx)
						if err != nil {
							return err
						}
						idx++
						if !opt[RUN_WITH_OPTION.RUN_WITHOUT_FILE_DESCRIPTOR] {
							// Set SOF to false before calling next
							// ctx = context.WithValue(ctx, sys_key.SOF, false)
							ictx.Set(sys_key.SOF, false)
						}
					}
					return nil
				}); err != nil {
					return nil, err
				}
			} else {
				for !comparator(prev, curr) {
					response, err = run(idx, ctx)
					if err != nil {
						return nil, err
					}
					idx++
					if !opt[RUN_WITH_OPTION.RUN_WITHOUT_FILE_DESCRIPTOR] {
						// Set SOF to false before calling next
						// ctx = context.WithValue(ctx, sys_key.SOF, false) // Update the context here
						ictx.Set(sys_key.SOF, false)
					}
				}

			}
			if !opt[RUN_WITH_OPTION.RUN_WITHOUT_FILE_DESCRIPTOR] {
				// ctx = context.WithValue(ctx, sys_key.EOF, "eof")
				ictx.Set(sys_key.EOF, "eof")
				if eofResponse, eofErr := next(ctx, nil); eofErr != nil {
					return nil, eofErr
				} else {
					response = eofResponse
				}
			}
			var errorOutcome error
			if len(errorCollection) > 0 {
				marshalled, _ := json.Marshal(errorCollection)
				errorOutcome = errors.New(string(marshalled))
			}
			return response, errorOutcome
		}
	}
}
