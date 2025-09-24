package inject

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	icontext "github.com/louvri/gokrt/context"
	"github.com/louvri/gokrt/option"
	"github.com/louvri/gokrt/sys_key"
)

func Middleware(
	e endpoint.Endpoint,
	modifier func(original any, data any, err error) (any, error),
	opts ...option.Option,
) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req any) (any, error) {
			var response, modified any
			var err error
			config := map[option.Option]bool{}
			if len(opts) > 0 {
				for _, opt := range opts {
					config[opt] = true
				}
			}
			if _, ok := ctx.Value(sys_key.GOKRT_CONTEXT).(*icontext.Context); !ok {
				ctx = icontext.New(ctx)
			}
			response, err = e(ctx, req)
			if err != nil && config[option.RUN_WITH_ERROR] {
				return nil, err
			}

			modified, err = modifier(req, response, err)
			if err != nil && config[option.RUN_WITH_ERROR] {
				return nil, err
			}

			return next(ctx, modified)
		}
	}
}
