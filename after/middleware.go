package after

import (
	"context"

	"github.com/go-kit/kit/endpoint"
)

func Middleware(e endpoint.Endpoint, preprocessor func(data interface{}, err error) interface{}) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			resp, err := next(ctx, req)
			if resp != nil || err != nil {

				result := preprocessor(resp, err)
				if result != nil {
					e(ctx, result)
				}

			}
			return resp, err
		}
	}
}
