package alter_test

import (
	"context"
	"errors"
	"log"
	"testing"

	"github.com/go-kit/kit/endpoint"
	"github.com/louvri/gokrt/after"
	"github.com/louvri/gokrt/alter"
	RUN_WITH_OPTION "github.com/louvri/gokrt/option"
)

type Mock interface {
	Main(ctx context.Context, request any) (any, error)
	First(ctx context.Context, request any) (any, error)
	Second(ctx context.Context, request any) (any, error)
	Third(ctx context.Context, request any) (any, error)
	Error(ctx context.Context, request any) (any, error)
}
type mock struct {
	logger *log.Logger
}

func NewMock() Mock {
	return &mock{
		logger: log.Default(),
	}
}

func (m *mock) Main(ctx context.Context, request any) (any, error) {
	return "main endpoint", nil
}

func (m *mock) First(ctx context.Context, request any) (any, error) {
	return "first endpoint", nil
}
func (m *mock) Second(ctx context.Context, request any) (any, error) {
	return "second endpoint", nil
}
func (m *mock) Third(ctx context.Context, request any) (any, error) {
	return "third endpoint", nil
}
func (m *mock) Error(ctx context.Context, request any) (any, error) {
	return nil, errors.New("it's error")
}
func TestHappyCaseAlter(t *testing.T) {
	m := NewMock()
	resp, _ := endpoint.Chain(
		alter.Middleware(m.First, func(data any, err error) any {
			t.Log(data)
			return data
		}, func(original, data any, err error) (any, error) {
			t.Log(data)
			return data, nil
		}),
		alter.Middleware(m.Second, func(data any, err error) any {
			t.Log(data)
			return data
		}, func(original, data any, err error) (any, error) {
			t.Log(data)
			return data, nil
		}),
		alter.Middleware(m.Third, func(data any, err error) any {
			t.Log(data)
			return data
		}, func(original, data any, err error) (any, error) {
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
		alter.Middleware(m.First, func(data any, err error) any {
			t.Log(data)
			return data
		}, func(original, data any, err error) (any, error) {
			t.Log(data)
			return data, nil
		}),
		alter.Middleware(m.Second, func(data any, err error) any {
			t.Log(data)
			return data
		}, func(original, data any, err error) (any, error) {
			t.Log(data)
			return data, nil
		}),
		alter.Middleware(m.Third, func(data any, err error) any {
			t.Log(data)
			return data
		}, func(original, data any, err error) (any, error) {
			t.Log(data)
			return data, nil
		}),
		alter.Middleware(m.Error, func(data any, err error) any {
			t.Log(data)
			return data
		}, func(original, data any, err error) (any, error) {
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
		alter.Middleware(m.First, func(data any, err error) any {
			if err != nil {
				return err
			}
			return data
		}, func(original, data any, err error) (any, error) {
			return data, err
		}),

		alter.Middleware(m.Error, func(data any, err error) any {
			if err != nil {
				return err
			}
			return data
		}, func(original, data any, err error) (any, error) {
			return data, err
		}),

		alter.Middleware(m.Third, func(data any, err error) any {
			if err != nil {
				return err
			}
			return data
		}, func(original, data any, err error) (any, error) {
			return data, err
		}),
	)(m.Main)(context.Background(), "")
	if err == nil || resp != nil {
		t.Log("should return error and response must nil")
		t.FailNow()
	}
}

func TestBeforeRun(t *testing.T) {
	m := NewMock()
	resp, _ := endpoint.Chain(
		alter.Middleware(m.First, func(data any, err error) any {
			t.Log(data)
			return data
		}, func(original, data any, err error) (any, error) {
			t.Log(original)
			return original, nil
		}, RUN_WITH_OPTION.EXECUTE_BEFORE),
		alter.Middleware(m.Second, func(data any, err error) any {
			t.Log(data)
			return data
		}, func(original, data any, err error) (any, error) {
			t.Log(data)
			return data, nil
		}),
		alter.Middleware(m.Third, func(data any, err error) any {
			t.Log(data)
			return data
		}, func(original, data any, err error) (any, error) {
			t.Log(data)
			return data, nil
		}),
	)(m.Main)(context.Background(), "")
	if result, ok := resp.(string); ok {
		if result != "second endpoint" {
			t.Logf("got %s but it should be 'second endpoint'", result)
			t.FailNow()
		}
	}
}

func TestAlterAfter(t *testing.T) {
	m := NewMock()
	_, err := endpoint.Chain(
		after.Middleware(m.First, func(data any, err error) any {
			return data
		}, nil),
		alter.Middleware(m.Main, func(data any, err error) any {
			t.Log(data)
			return data
		}, func(original, data any, err error) (any, error) {
			t.Log(data)
			return data, nil
		}),
	)(m.Error)(context.Background(), "")
	if err == nil {
		t.Log("should error")
		t.FailNow()
	}
}
