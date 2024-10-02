package inject

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/louvri/gokrt/option"
)

func Middleware(
	e endpoint.Endpoint,
	preprocessor func(data interface{}) interface{},
	postprocessor func(original interface{}, data interface{}, err error) (interface{}, error),
	opts ...option.Option,
) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			var response, original interface{}
			var err error
			config := map[option.Option]bool{}
			if len(opts) > 0 {
				for _, opt := range opts {
					config[opt] = true
				}
			}
			original = preprocessor(req)

			response, err = e(ctx, original)
			if err != nil && config[option.RUN_WITH_ERROR] {
				return nil, err
			}

			req, err = postprocessor(original, response, err)
			if err != nil && config[option.RUN_WITH_ERROR] {
				return nil, err
			}
			return next(ctx, req)
		}
	}
}
