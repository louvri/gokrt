package alter

import (
	"context"
	"sync"

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
					var wg sync.WaitGroup
					var altered interface{}
					wg.Add(1)
					go func() {
						defer wg.Done()
						altered, err = e(ctx, result)
					}()
					wg.Wait()
					return postprocessor(original, altered, err)
				} else {
					return nil, nil
				}
			}
			return original, nil

		}
	}
}
