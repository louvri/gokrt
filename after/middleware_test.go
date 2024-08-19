package after_test

import (
	"context"
	"errors"
	"log"
	"testing"

	"github.com/go-kit/kit/endpoint"
	"github.com/louvri/gokrt/after"
	RUN_WITH_OPTION "github.com/louvri/gokrt/model/option"
)

type Mock interface {
	Main(ctx context.Context, request interface{}) (interface{}, error)
	First(ctx context.Context, request interface{}) (interface{}, error)
	Second(ctx context.Context, request interface{}) (interface{}, error)
	Third(ctx context.Context, request interface{}) (interface{}, error)
	Error(ctx context.Context, request interface{}) (interface{}, error)
}

var EXPECTED_RESULT string = "main endpoint"
var ERROR error = errors.New("it's error")

type mock struct {
	logger *log.Logger
}

func NewMock() Mock {
	return &mock{
		logger: log.Default(),
	}
}

func (m *mock) Main(ctx context.Context, request interface{}) (interface{}, error) {
	return EXPECTED_RESULT, nil
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
	return nil, ERROR
}
func TestHappyCaseAlter(t *testing.T) {
	m := NewMock()
	resp, _ := endpoint.Chain(
		after.Middleware(m.First, func(data interface{}, err error) interface{} {
			t.Log(data)
			return data
		}),
		after.Middleware(m.Second, func(data interface{}, err error) interface{} {
			t.Log(data)
			return data
		}),
		after.Middleware(m.Third, func(data interface{}, err error) interface{} {
			t.Log(data)
			return data
		}),
	)(m.Main)(context.Background(), "")
	if result, ok := resp.(string); ok {
		if result != EXPECTED_RESULT {
			t.Logf("got '%s' expected '%s'", EXPECTED_RESULT, result)
			t.FailNow()
		}
	}
}

func TestNotStopWithError(t *testing.T) {
	m := NewMock()
	resp, _ := endpoint.Chain(
		after.Middleware(m.First, func(data interface{}, err error) interface{} {
			t.Log(data)
			return data
		}),
		after.Middleware(m.Second, func(data interface{}, err error) interface{} {
			t.Log(err)
			return err
		}),
		after.Middleware(m.Error, func(data interface{}, err error) interface{} {
			t.Log(data)
			return data
		}, RUN_WITH_OPTION.RUN_WITH_ERROR),
	)(m.Main)(context.Background(), "")
	if result, ok := resp.(string); ok {
		if result != EXPECTED_RESULT {
			t.Logf("got '%s' expected '%s'", EXPECTED_RESULT, result)
			t.FailNow()
		}
	}
}

func TestStopWithError(t *testing.T) {
	m := NewMock()
	resp, _ := endpoint.Chain(
		after.Middleware(m.First, func(data interface{}, err error) interface{} {
			t.Log(data)
			return data
		}),
		after.Middleware(m.Second, func(data interface{}, err error) interface{} {
			t.Log(err)
			return err
		}),
		after.Middleware(m.Error, func(data interface{}, err error) interface{} {
			t.Log(data)
			return data
		}),
	)(m.Error)(context.Background(), "")
	if result, ok := resp.(string); ok {
		if result != ERROR.Error() {
			t.Logf("got '%s' expected '%s'", ERROR.Error(), result)
			t.FailNow()
		}
	}
}
