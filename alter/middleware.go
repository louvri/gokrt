package alter

import (
	"context"

	"github.com/go-kit/kit/endpoint"
)

func Middleware(
	e endpoint.Endpoint,
	preprocessor func(data interface{}, err error) interface{},
	postprocessor func(original interface{}, data interface{}, err error) (interface{}, error)) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			original, err := next(ctx, req)
			if original != nil || err != nil {
				result := preprocessor(original, err)
				if result != nil {
					var altered interface{}
					altered, err = e(ctx, result)
					return postprocessor(original, altered, err)
				} else {
					return nil, nil
				}
			}
			return original, nil

		}
	}
}
