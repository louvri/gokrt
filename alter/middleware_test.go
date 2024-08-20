package alter_test

import (
	"context"
	"errors"
	"log"
	"testing"

	"github.com/go-kit/kit/endpoint"
	"github.com/louvri/gokrt/alter"
	RUN_WITH_OPTION "github.com/louvri/gokrt/option"
)

type Mock interface {
	Main(ctx context.Context, request interface{}) (interface{}, error)
	First(ctx context.Context, request interface{}) (interface{}, error)
	Second(ctx context.Context, request interface{}) (interface{}, error)
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
	return "main endpoint", nil
}

func (m *mock) First(ctx context.Context, request interface{}) (interface{}, error) {
	return "first endpoint", nil
}
func (m *mock) Second(ctx context.Context, request interface{}) (interface{}, error) {
	return "second endpoint", nil
}
func (m *mock) Third(ctx context.Context, request interface{}) (interface{}, error) {
	return "third endpoint", nil
}
func (m *mock) Error(ctx context.Context, request interface{}) (interface{}, error) {
	return nil, errors.New("it's error")
}
func TestHappyCaseAlter(t *testing.T) {
	m := NewMock()
	resp, _ := endpoint.Chain(
		alter.Middleware(m.First, func(data interface{}, err error) interface{} {
			t.Log(data)
			return data
		}, func(original, data interface{}, err error) (interface{}, error) {
			t.Log(data)
			return data, nil
		}),
		alter.Middleware(m.Second, func(data interface{}, err error) interface{} {
			t.Log(data)
			return data
		}, func(original, data interface{}, err error) (interface{}, error) {
			t.Log(data)
			return data, nil
		}),
		alter.Middleware(m.Third, func(data interface{}, err error) interface{} {
			t.Log(data)
			return data
		}, func(original, data interface{}, err error) (interface{}, error) {
			t.Log(data)
			return data, nil
		}),
	)(m.Main)(context.Background(), "")
	if result, ok := resp.(string); ok {
		if result != "first endpoint" {
			t.Logf("got '%s' expected 'first endpoint'", result)
			t.FailNow()
		}
	}
}

func TestNotStopWithError(t *testing.T) {
	m := NewMock()

	resp, err := endpoint.Chain(
		alter.Middleware(m.First, func(data interface{}, err error) interface{} {
			t.Log(data)
			return data
		}, func(original, data interface{}, err error) (interface{}, error) {
			t.Log(data)
			return data, nil
		}),
		alter.Middleware(m.Second, func(data interface{}, err error) interface{} {
			t.Log(data)
			return data
		}, func(original, data interface{}, err error) (interface{}, error) {
			t.Log(data)
			return data, nil
		}),
		alter.Middleware(m.Third, func(data interface{}, err error) interface{} {
			t.Log(data)
			return data
		}, func(original, data interface{}, err error) (interface{}, error) {
			t.Log(data)
			return data, nil
		}),
		alter.Middleware(m.Error, func(data interface{}, err error) interface{} {
			t.Log(data)
			return data
		}, func(original, data interface{}, err error) (interface{}, error) {
			t.Log(data)
			return data, nil
		}, RUN_WITH_OPTION.RUN_WITH_ERROR),
	)(m.Main)(context.Background(), "")
	if result, ok := resp.(string); ok {
		if result != "first endpoint" {
			t.Logf("got '%s' expected 'first endpoint'", result)
			t.FailNow()
		}
	}
	if err != nil {
		t.Log(err.Error())
		t.FailNow()
	}
}

func TestStopWithError(t *testing.T) {
	m := NewMock()

	resp, err := endpoint.Chain(
		alter.Middleware(m.First, func(data interface{}, err error) interface{} {
			if err != nil {
				return err
			}
			return data
		}, func(original, data interface{}, err error) (interface{}, error) {
			return data, err
		}),

		alter.Middleware(m.Error, func(data interface{}, err error) interface{} {
			if err != nil {
				return err
			}
			return data
		}, func(original, data interface{}, err error) (interface{}, error) {
			return data, err
		}),

		alter.Middleware(m.Third, func(data interface{}, err error) interface{} {
			if err != nil {
				return err
			}
			return data
		}, func(original, data interface{}, err error) (interface{}, error) {
			return data, err
		}),
	)(m.Main)(context.Background(), "")
	if err == nil || resp != nil {
		t.Log("should return error and response must nil")
		t.FailNow()
	}
}
