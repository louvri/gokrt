package loop_array

import (
	"context"
	"sync"

	"github.com/go-kit/kit/endpoint"
)

func Middleware(e endpoint.Endpoint, preprocessor func(data interface{}) interface{}, wait ...bool) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			resp, err := next(ctx, req)
			if err != nil {
				return resp, err
			} else {
				if resp != nil {
					if len(wait) > 0 && wait[0] {
						var wg sync.WaitGroup
						wg.Add(1)
						go func() {
							if arr, ok := resp.([]map[string]interface{}); ok {
								for _, item := range arr {
									e(ctx, preprocessor(item))
								}
							} else if arr, ok := resp.([]interface{}); ok {
								for _, item := range arr {
									e(ctx, preprocessor(item))
								}
							}
							wg.Done()
						}()
						wg.Wait()
					} else {
						if arr, ok := resp.([]map[string]interface{}); ok {
							for _, item := range arr {
								e(ctx, preprocessor(item))
							}
						} else if arr, ok := resp.([]interface{}); ok {
							for _, item := range arr {
								e(ctx, preprocessor(item))
							}
						}
					}
				}
				return resp, nil
			}
		}
	}
}
