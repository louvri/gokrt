package meta

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	icontext "github.com/louvri/gokrt/context"
	"github.com/louvri/gokrt/sys_key"
)

func Middleware(compiler func(response any, err error)) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req any) (any, error) {
			resp, err := next(ctx, req)
			var ictx *icontext.Context
			if _, ok := ctx.Value(sys_key.GOKRT_CONTEXT).(*icontext.Context); !ok {
				ctx = icontext.New(ctx)
			}
			ictx = ctx.Value(sys_key.GOKRT_CONTEXT).(*icontext.Context)
			eof := ictx.Get(sys_key.EOF)
			if eof == nil || eof != "eof" {
				compiler(resp, err)
			}
			return resp, err
		}
	}
}
