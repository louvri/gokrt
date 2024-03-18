package array

import (
	"context"
	"sync"

	"github.com/go-kit/kit/endpoint"
)

func Middleware(e endpoint.Endpoint, preprocessor func(data interface{}) interface{}) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			resp, err := next(ctx, req)
			if err != nil {
				return resp, err
			} else {
				if resp != nil {
					if arr, ok := resp.([]map[string]interface{}); ok {
						var wg sync.WaitGroup
						wg.Add(1)
						go func() {
							for _, item := range arr {
								e(ctx, preprocessor(item))
							}
							wg.Done()
						}()
					} else if arr, ok := resp.([]interface{}); ok {
						var wg sync.WaitGroup
						wg.Add(1)
						go func() {
							for _, item := range arr {
								e(ctx, preprocessor(item))
							}
							wg.Done()
						}()
					}
				}
				return resp, nil
			}
		}
	}
}
