package meta

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/louvri/gokrt/sys_key"
)

func Middleware(compiler func(response interface{}, err error)) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			resp, err := next(ctx, req)
			eof := ctx.Value(sys_key.EOF)
			if eof == nil || eof != "eof" {
				compiler(resp, err)
			}
			return resp, err
		}
	}
}
