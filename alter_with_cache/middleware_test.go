package alter_with_cache_test

import (
	"context"
	"errors"
	"log"
	"testing"

	"github.com/go-kit/kit/endpoint"
	"github.com/louvri/gokrt/alter_with_cache"
	"github.com/louvri/gokrt/cache"
)

type Mock interface {
	Loop(ctx context.Context, request any) (any, error)
	Main(ctx context.Context, request any) (any, error)
	First(ctx context.Context, request any) (any, error)
	Alter(ctx context.Context, request any) (any, error)
	Third(ctx context.Context, request any) (any, error)
	Error(ctx context.Context, request any) (any, error)
	Forth(ctx context.Context, request any) (any, error)
}
type mock struct {
	logger *log.Logger
}

func NewMock() Mock {
	return &mock{
		logger: log.Default(),
	}
}

type Data struct {
	Cache    map[string]any
	Request  string
	Altered  string
	Response string
	Status   string
}

func (m *mock) Main(ctx context.Context, request any) (any, error) {
	return "main", nil
}

func (m *mock) First(ctx context.Context, request any) (any, error) {
	return "first inject endpoint", nil
}
func (m *mock) Forth(ctx context.Context, request any) (any, error) {
	return "forth", nil
}
func (m *mock) Alter(ctx context.Context, request any) (any, error) {
	if result, ok := request.(map[string]any); ok {
		result["status"] = "injected"
		return result, nil
	} else if result, ok := request.(Data); ok {
		result.Status = "injected"
		return result, nil
	}
	return "alter inject endpoint", nil
}
func (m *mock) Third(ctx context.Context, request any) (any, error) {
	return "third inject endpoint", nil
}
func (m *mock) Error(ctx context.Context, request any) (any, error) {
	return nil, errors.New("it's error")
}

func (m *mock) Loop(ctx context.Context, request any) (any, error) {
	return []any{
		"main1",
		"main2",
		"main3",
		"main4",
		"main5",
		"main6",
	}, nil
}

func TestAlterWithCache(t *testing.T) {
	m := NewMock()

	result, err := endpoint.Chain(
		cache.Middleware(m.First, func(req any) any {
			return "req 1"
		}, "key-1"),
		cache.Middleware(m.Third, func(req any) any {
			return "req 1"
		}, "key-2"),
<<<<<<< HEAD
		alter_with_cache.Middleware(func(data any, err error) any {
			return data
		}, func(original, data, cache any, err error) (any, error) {
			var injected Data

			if cached, ok := cache.(map[string]any); ok {
				injected.Cache = cached
				injected.Response = data.(string)
				injected.Status = "injected"
			}

			return injected, err
		}),
	)(m.Main)(context.Background(), Data{
=======
		loop_array.Middleware(
			endpoint.Chain(
				alter_with_cache.Middleware(func(data any, err error) any {
					return "alter1"
				}, func(original, data, cache any, err error) (any, error) {
					return "alter1", nil
				}),
			)(m.Third), func(data any, err error) any {
				return data
			}, nil,
		),
	)(m.Loop)(context.Background(), Data{
>>>>>>> main
		Request: "request",
	})

	if err != nil {
		t.Log(err.Error())
		t.FailNow()
	}
	if r, ok := result.(Data); ok {
		if len(r.Cache) != 2 {
			t.Log("cache len should 2 since injected twitce")
			t.FailNow()
		}

		if r.Response != "main" {
			t.Log("should return 'main' result since it's saved result from maind endpoint")
			t.FailNow()
		}

		if r.Status != "injected" {
			t.Log("should return status 'injected' result since it's saved result from alter endpoint")
			t.FailNow()
		}
	} else {
		t.Log("invalid data")
		t.FailNow()
	}
}
