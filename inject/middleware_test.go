package inject_test

import (
	"context"
	"errors"
	"log"
	"testing"

	"github.com/go-kit/kit/endpoint"
	"github.com/louvri/gokrt/inject"
)

type Mock interface {
	Main(ctx context.Context, request interface{}) (interface{}, error)
	First(ctx context.Context, request interface{}) (interface{}, error)
	Inject(ctx context.Context, request interface{}) (interface{}, error)
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

type Data struct {
	Request  string
	Status   string
	Injected []string
}

func (m *mock) Main(ctx context.Context, request interface{}) (interface{}, error) {
	if data, ok := request.(Data); ok {
		if len(data.Injected) >= 2 && data.Status != "finale" {
			return nil, errors.New("result should be finale")
		}

		if len(data.Injected) == 1 && data.Status != "injected" {
			return nil, errors.New("result should be injected")
		}

		if data.Request != "initiated request" {
			return nil, errors.New("request should be same as pre declared before")
		}

	}
	return "main", nil
}

func (m *mock) First(ctx context.Context, request interface{}) (interface{}, error) {
	return "first inject endpoint", nil
}
func (m *mock) Inject(ctx context.Context, request interface{}) (interface{}, error) {
	if result, ok := request.(map[string]interface{}); ok {
		result["status"] = "injected"
		return result, nil
	} else if result, ok := request.(Data); ok {
		result.Status = "injected"
		if len(result.Injected) > 0 {
			result.Status = "finale"
		}
		result.Injected = append(result.Injected, "injected from alter endpoint")
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

func TestInjection(t *testing.T) {
	m := NewMock()

	result, err := endpoint.Chain(
		inject.Middleware(m.Inject, func(original, data interface{}, err error) (interface{}, error) {
			return data, nil
		}),
		inject.Middleware(m.Inject, func(original, data interface{}, err error) (interface{}, error) {
			return data, nil
		}),
	)(m.Main)(context.Background(), Data{
		Request: "initiated request",
	})
	if err != nil {
		t.Log(err.Error())
		t.FailNow()
	}

	if res, ok := result.(string); ok {
		if res != "main" {
			t.Log("result should be main")
			t.FailNow()
		}
	} else if !ok || res == "" {
		if res != "main" {
			t.Log("result should be main")
			t.FailNow()
		}
	}
}
