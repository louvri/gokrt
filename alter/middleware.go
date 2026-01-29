package alter

import (
	"context"
	"sync"

	"github.com/go-kit/kit/endpoint"
	icontext "github.com/louvri/gokrt/context"
	RUN_WITH_OPTION "github.com/louvri/gokrt/option"
)

func Middleware(
	e endpoint.Endpoint,
	preprocessor func(data any, err error) any,
	postprocessor func(original any, data any, err error) (any, error),
	opts ...RUN_WITH_OPTION.Option) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req any) (any, error) {
			var ictx icontext.Icontext
			if tmp, ok := ctx.(icontext.Icontext); ok {
				ictx = tmp
			} else {
				ictx = icontext.New(ctx)
			}
			opt := make(map[RUN_WITH_OPTION.Option]bool)
			for _, option := range opts {
				opt[option] = true
			}
			var original any
			var err error
			runOnError := opt[RUN_WITH_OPTION.RUN_WITH_ERROR]
			original, err = next(ictx, req)
			if err != nil && !runOnError {
				return original, err
			}

			if original != nil {
				result := original
				if preprocessor != nil {
					result = preprocessor(original, err)
				}
				if result != nil {
					var altered any
					var wg sync.WaitGroup
					wg.Add(1)
					go func() {
						defer wg.Done()
						altered, err = e(ictx, result)
					}()
					wg.Wait()
					if postprocessor == nil {
						return altered, err
					}
					return postprocessor(original, altered, err)
				} else {
					return nil, nil
				}
			}
			return original, err

		}
	}
}
