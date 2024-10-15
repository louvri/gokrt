package inject_with_cache_test

import (
	"context"
	"errors"
	"fmt"
	"log"
	"testing"

	"github.com/go-kit/kit/endpoint"
	"github.com/louvri/gokrt/cache"
	"github.com/louvri/gokrt/inject_with_cache"
)

type Mock interface {
	Main(ctx context.Context, request interface{}) (interface{}, error)
	First(ctx context.Context, request interface{}) (interface{}, error)
	Alter(ctx context.Context, request interface{}) (interface{}, error)
	Third(ctx context.Context, request interface{}) (interface{}, error)
	Error(ctx context.Context, request interface{}) (interface{}, error)
}
type mock struct {
	logger *log.Logger
}

func NewMock() Mock {
	return &mock{
		logger: log.Default(),
	}
}

func (m *mock) Main(ctx context.Context, request interface{}) (interface{}, error) {
	return request, nil
}

func (m *mock) First(ctx context.Context, request interface{}) (interface{}, error) {
	return "first inject endpoint", nil
}
func (m *mock) Alter(ctx context.Context, request interface{}) (interface{}, error) {
	if result, ok := request.(map[string]interface{}); ok {
		result["status"] = "injected"
		return result, nil
	}
	return "alter inject endpoint", nil
}
func (m *mock) Third(ctx context.Context, request interface{}) (interface{}, error) {
	return "third inject endpoint", nil
}
func (m *mock) Error(ctx context.Context, request interface{}) (interface{}, error) {
	return nil, errors.New("it's error")
}

func TestAlterCache(t *testing.T) {
	m := NewMock()
	resp, err := endpoint.Chain(
		cache.Middleware(m.First, func(req interface{}) interface{} {
			return nil
		}),
		inject_with_cache.Middleware(func(cache, data interface{}) interface{} {
			if data == nil {
				return cache.(string)
			}
			return fmt.Sprintf("%s + %s", cache.(string), data.(string))
		}),
	)(m.Main)(context.Background(), "request")
	if err != nil {
		t.Log(err.Error())
		t.FailNow()
	}

	if result, ok := resp.(map[string]interface{}); ok && result != nil {
		if result["data"] == nil {
			t.Log("data field in result shouldn't be null")
			t.FailNow()
		}

		if result["cache"] == nil {
			t.Log("cache field in result shouldn't be null")
			t.FailNow()
		}

		if result["original"] == nil {
			t.Log("original field in result shouldn't be null")
			t.FailNow()
		}
	}
}

func TestAlterMultipleCache(t *testing.T) {
	m := NewMock()
	var tobeProcessed map[string]interface{}
	_, err := endpoint.Chain(
		cache.Middleware(m.First, func(req interface{}) interface{} {
			return nil
		}, "cache-1"),
		cache.Middleware(m.Third, func(req interface{}) interface{} {
			return nil
		}, "cache-2"),
		inject_with_cache.Middleware(func(cache, data interface{}) interface{} {
			tobeProcessed = cache.(map[string]interface{})
			return tobeProcessed
		}),
	)(m.Main)(context.Background(), "request")
	if err != nil {
		t.Log(err.Error())
		t.FailNow()
	}
	if len(tobeProcessed) == 0 {
		t.Log("data field in result shouldn't be null")
		t.FailNow()
	}
	t.Log(tobeProcessed)
}
