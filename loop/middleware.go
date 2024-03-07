package loop

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kit/kit/endpoint"
	"github.com/louvri/gokrt/sys_key"
)

// loop
func Middleware(
	comparator func(prev, curr interface{}) bool,
	modifier func(req interface{}, next interface{}) interface{},
	ignoreError bool) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			eof := ctx.Value(sys_key.EOF)
			if eof != nil && eof == "eof" {
				return next(ctx, nil)
			} else {
				var err error
				var prev, curr interface{}
				curr = make([]map[string]interface{}, 0)
				modifiedReq := req
				ctx = context.WithValue(ctx, sys_key.SOF, true)
				for !comparator(prev, curr) {
					modifiedReq := modifier(modifiedReq, curr)
					prev = curr
					ctx = context.WithValue(ctx, sys_key.DATA_REF, prev)
					curr, err = next(ctx, modifiedReq)
					if !ignoreError && err != nil {
						ctx = context.WithValue(ctx, sys_key.EOF, "eof")
						next(ctx, nil)
						return nil, fmt.Errorf("%s:%s", "loop_middleware:", err.Error())
					}
					ctx = context.WithValue(ctx, sys_key.SOF, false)
					time.Sleep(0)
				}
				ctx = context.WithValue(ctx, sys_key.EOF, "eof")
				if eofResponse, eofErr := next(ctx, nil); eofErr != nil {
					return nil, eofErr
				} else {
					return eofResponse, err
				}
			}
		}
	}
}
