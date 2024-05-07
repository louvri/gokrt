package after

import (
	"context"
	"sync"
	"time"

	"github.com/go-kit/kit/endpoint"
	icontext "github.com/louvri/gokrt/icontext"
)

func Middleware(e endpoint.Endpoint, preprocessor func(data interface{}, err error) interface{}, wait ...bool) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			resp, err := next(ctx, req)
			if resp != nil || err != nil {
				result := preprocessor(resp, err)
				if result != nil {
					nwait := len(wait)
					if nwait > 0 && wait[0] {
						var wg sync.WaitGroup
						wg.Add(1)
						go func() {
							e(ctx, result)
							wg.Done()
						}()
						wg.Wait()
					} else {
						ctx = icontext.New(ctx, time.Now().Add(1*time.Second))
						go e(ctx, result)
					}
				}
			}
			return resp, err
		}
	}
}
