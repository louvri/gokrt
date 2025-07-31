package alter_with_cache

import (
	"context"
	"sync"

	"github.com/go-kit/kit/endpoint"
	RUN_WITH_OPTION "github.com/louvri/gokrt/option"
	"github.com/louvri/gokrt/sys_key"
)

func Middleware(
	preprocessor func(data any, err error) any,
	postprocessor func(original, data, cache any, err error) (any, error),
	opts ...RUN_WITH_OPTION.Option) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req any) (any, error) {
			opt := make(map[RUN_WITH_OPTION.Option]bool)
			for _, option := range opts {
				opt[option] = true
			}
			original := req
			var err error

			inmem := ctx.Value(sys_key.CACHE_KEY)
			if inmemCache, ok := inmem.(map[string]any); ok {
				modified := preprocessor(original, err)
				if modified != nil {
					var result any
					if runAsync := opt[RUN_WITH_OPTION.RUN_ASYNC_WAIT]; runAsync {
						var wg sync.WaitGroup
						wg.Add(1)
						go func() {
							defer wg.Done()
							result, err = next(ctx, modified)
						}()
						wg.Wait()
					} else {
						result, err = next(ctx, modified)
					}
					if !opt[RUN_WITH_OPTION.RUN_WITH_ERROR] && err != nil {
						return nil, err
					}
					return postprocessor(original, result, inmemCache, err)
				}
				return nil, nil
			}
			if !opt[RUN_WITH_OPTION.RUN_WITH_ERROR] && err != nil {
				return nil, err
			}
			return original, err
		}
	}
}
