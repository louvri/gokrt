package test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/go-kit/kit/endpoint"
	"github.com/louvri/gokrt/after"
	"github.com/louvri/gokrt/alter"
	"github.com/louvri/gokrt/cache"
	"github.com/louvri/gokrt/on_eof"
	"github.com/louvri/gokrt/option"
	"github.com/louvri/gokrt/sys_key"
	"github.com/louvri/gokrt/use_cache"
)

type key int

var test key = 1

func TestAfter(t *testing.T) {
	ctx := context.WithValue(context.Background(), test, 1)
	response, err := endpoint.Chain(after.Middleware(
		func(ctx context.Context, req interface{}) (interface{}, error) {
			fmt.Printf("HELLOOO %v\n", ctx.Value(test))
			fmt.Println(req)
			time.Sleep(1000)
			return nil, nil
		},
		func(data interface{}, err error) interface{} {
			return data
		},
	))(func(ctx context.Context, req interface{}) (interface{}, error) {
		a := 0
		for i := 0; i < 10000; i++ {
			a++
		}
		return a, nil
	})(ctx, 3)
	if err != nil {
		t.Fatal(err.Error())
	}
	fmt.Println(response)
}

func TestOnEof(t *testing.T) {
	ctx := context.WithValue(context.Background(), sys_key.EOF, "eof")
	response, err := endpoint.Chain(
		on_eof.Middleware(
			alter.Middleware(
				func(ctx context.Context, req interface{}) (interface{}, error) {
					return "hello world 1", nil
				},
				func(data interface{}, err error) interface{} {
					return data
				},
				func(data1, data2 interface{}, err error) (interface{}, error) {
					return data2, nil
				},
			),
			after.Middleware(
				func(ctx context.Context, req interface{}) (interface{}, error) {
					return "hello world 2", nil
				},
				func(data interface{}, err error) interface{} {
					return data
				},
			),
		),
	)(func(ctx context.Context, req interface{}) (interface{}, error) {
		return "satu", nil
	})(ctx, -1)
	if response.(string) != "hello world 1" {
		t.Fatal("wrong result")
	}
	if err != nil {
		t.Fatal(err.Error())
	}
}
func TestOnEofWhileError(t *testing.T) {
	ctx := context.WithValue(context.Background(), sys_key.EOF, "err")
	response, err := endpoint.Chain(
		on_eof.Middleware(
			alter.Middleware(
				func(ctx context.Context, req interface{}) (interface{}, error) {
					return "hello world", nil
				},
				func(data interface{}, err error) interface{} {
					return data
				},
				func(data1, data2 interface{}, err error) (interface{}, error) {
					return data2, nil
				},
			),
		),
	)(func(ctx context.Context, req interface{}) (interface{}, error) {
		return "satu", nil
	})(ctx, -1)
	if response.(string) == "hello world" {
		t.Fatal("wrong result")
	}
	if err != nil {
		t.Fatal(err.Error())
	}
}

func TestMultipleCacheAndUseCache(t *testing.T) {
	ctx := context.Background()
	key1 := "key-1"
	cache1 := "cache-1"
	key2 := "key-2"
	cache2 := "cache-2"

	// request := make(map[string]interface{})
	_, err := endpoint.Chain(
		cache.Middleware(func(ctx context.Context, req interface{}) (interface{}, error) {
			return cache1, nil
		}, func(req interface{}) interface{} {
			return req
		}, option.Config{
			CacheKey: key1,
		}),
		cache.Middleware(func(ctx context.Context, req interface{}) (interface{}, error) {
			return cache2, nil
		}, func(req interface{}) interface{} {
			return req
		}, option.Config{
			CacheKey: key2,
		}),

		use_cache.Middleware(
			func(ctx context.Context, request interface{}) (response interface{}, err error) {
				return request, nil
			}, func(cache, next interface{}) interface{} {
				return fmt.Sprintf("%v + %v", next, cache)
			},
		),
	)(func(ctx context.Context, req interface{}) (interface{}, error) {
		return "main result", nil
	})(ctx, "current request")
	if err != nil {
		t.Log(err.Error())
		t.FailNow()
	}

}

func TestSingleCacheAndUseCache(t *testing.T) {
	ctx := context.Background()
	cache1 := "cache-1"

	// request := make(map[string]interface{})
	_, err := endpoint.Chain(
		cache.Middleware(func(ctx context.Context, req interface{}) (interface{}, error) {
			return cache1, nil
		}, func(req interface{}) interface{} {
			return req
		}),
		use_cache.Middleware(
			func(ctx context.Context, request interface{}) (response interface{}, err error) {
				return request, nil
			}, func(cache, next interface{}) interface{} {
				return fmt.Sprintf("%v + %v", next, cache)
			},
		),
	)(func(ctx context.Context, req interface{}) (interface{}, error) {
		return "main result", nil
	})(ctx, "current request")
	if err != nil {
		t.Log(err.Error())
		t.FailNow()
	}
}

func TestMultipleCacheEmptyOneAndUseCache(t *testing.T) {
	ctx := context.Background()
	// key1 := "key-1"
	cache1 := "cache-1"
	key2 := "key-2"
	cache2 := "cache-2"

	// request := make(map[string]interface{})
	_, err := endpoint.Chain(
		cache.Middleware(func(ctx context.Context, req interface{}) (interface{}, error) {
			return cache1, nil
		}, func(req interface{}) interface{} {
			return req
		}),
		cache.Middleware(func(ctx context.Context, req interface{}) (interface{}, error) {
			return cache2, nil
		}, func(req interface{}) interface{} {
			return req
		}, option.Config{
			CacheKey: key2,
		}),
		use_cache.Middleware(
			func(ctx context.Context, request interface{}) (response interface{}, err error) {
				return request, nil
			}, func(cache, next interface{}) interface{} {
				return fmt.Sprintf("%v + %v", next, cache)
			},
		),
	)(func(ctx context.Context, req interface{}) (interface{}, error) {
		return "main result", nil
	})(ctx, "current request")
	if err != nil {
		t.Log(err.Error())
		t.FailNow()
	}

}

func TestEmptyEndpointUseCache(t *testing.T) {
	ctx := context.Background()
	key1 := "key-1"
	cache1 := "cache-1"
	key2 := "key-2"
	cache2 := "cache-2"

	// request := make(map[string]interface{})
	_, err := endpoint.Chain(
		cache.Middleware(func(ctx context.Context, req interface{}) (interface{}, error) {
			return cache1, nil
		}, func(req interface{}) interface{} {
			return req
		}, option.Config{
			CacheKey: key1,
		}),
		cache.Middleware(func(ctx context.Context, req interface{}) (interface{}, error) {
			return cache2, nil
		}, func(req interface{}) interface{} {
			return req
		}, option.Config{
			CacheKey: key2,
		}),
		use_cache.Middleware(
			nil, func(cache, next interface{}) interface{} {
				return fmt.Sprintf("%v + %v", next, cache)
			}, option.Config{
				Option: []option.Option{option.EXECUTE_BEFORE},
			},
		),
	)(func(ctx context.Context, req interface{}) (interface{}, error) {
		return "main result", nil
	})(ctx, "current request")
	if err != nil {
		t.Log(err.Error())
		t.FailNow()
	}

}

func TestNotEmptyEndpointUseCache(t *testing.T) {
	ctx := context.Background()
	key1 := "key-1"
	cache1 := "cache-1"
	key2 := "key-2"
	cache2 := "cache-2"

	// request := make(map[string]interface{})
	_, err := endpoint.Chain(
		cache.Middleware(func(ctx context.Context, req interface{}) (interface{}, error) {
			return cache1, nil
		}, func(req interface{}) interface{} {
			return req
		}, option.Config{
			CacheKey: key1,
		}),
		cache.Middleware(func(ctx context.Context, req interface{}) (interface{}, error) {
			return cache2, nil
		}, func(req interface{}) interface{} {
			return req
		}, option.Config{
			CacheKey: key2,
		}),
		use_cache.Middleware(
			func(ctx context.Context, request interface{}) (response interface{}, err error) {
				return request, nil
			}, func(cache, next interface{}) interface{} {
				return fmt.Sprintf("%v + %v", next, cache)
			},
		),
	)(func(ctx context.Context, req interface{}) (interface{}, error) {
		return "main result", nil
	})(ctx, "current request")
	if err != nil {
		t.Log(err.Error())
		t.FailNow()
	}
}

func TestCacheAndUseCache(t *testing.T) {
	ctx := context.Background()
	cache1 := "cache-1"

	// request := make(map[string]interface{})
	_, err := endpoint.Chain(
		cache.Middleware(func(ctx context.Context, req interface{}) (interface{}, error) {
			return cache1, nil
		}, func(req interface{}) interface{} {
			return req
		}),
		use_cache.Middleware(
			func(ctx context.Context, request interface{}) (response interface{}, err error) {
				return request, nil
			}, func(cache, next interface{}) interface{} {
				return fmt.Sprintf("%v + %v", next, cache)
			},
		),
	)(func(ctx context.Context, req interface{}) (interface{}, error) {
		return "main result", nil
	})(ctx, "current request")
	if err != nil {
		t.Log(err.Error())
		t.FailNow()
	}

}

func TestCacheAndUseCacheWithOptionsOnly(t *testing.T) {
	ctx := context.Background()
	cache1 := "cache-1"

	// request := make(map[string]interface{})
	_, err := endpoint.Chain(
		cache.Middleware(func(ctx context.Context, req interface{}) (interface{}, error) {
			return cache1, nil
		}, func(req interface{}) interface{} {
			return req
		}, option.Config{
			Option: []option.Option{option.EXECUTE_BEFORE},
		}),
		use_cache.Middleware(
			func(ctx context.Context, request interface{}) (response interface{}, err error) {
				return request, nil
			}, func(cache, next interface{}) interface{} {
				return fmt.Sprintf("%v + %v", next, cache)
			}, option.Config{
				Option: []option.Option{option.EXECUTE_BEFORE},
			},
		),
	)(func(ctx context.Context, req interface{}) (interface{}, error) {
		return "main result", nil
	})(ctx, "current request")
	if err != nil {
		t.Log(err.Error())
		t.FailNow()
	}

}
