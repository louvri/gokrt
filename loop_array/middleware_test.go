package loop_array_test

import (
	"context"
	"errors"
	"log"
	"strings"
	"testing"

	"github.com/go-kit/kit/endpoint"
	"github.com/louvri/gokrt/loop_array"
	"github.com/louvri/gokrt/option"
)

var err = errors.New("error appear")

type Mock interface {
	Main(ctx context.Context, request interface{}) (interface{}, error)
	Executor(ctx context.Context, request interface{}) (interface{}, error)
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
	return []interface{}{
		"1stIndex",
		"2ndIndex",
		"3rdIndex",
		"4thIndex",
		err,
		"5thIndex",
	}, nil
}

func (m *mock) Executor(ctx context.Context, request interface{}) (interface{}, error) {
	m.logger.Printf("output exector endpoint: %s", request)
	if err, ok := request.(error); ok && err != nil {
		return nil, err
	}
	return request, nil
}

func TestLoopArrayWithError(t *testing.T) {
	m := NewMock()
	_, r := endpoint.Chain(
		loop_array.Middleware(
			m.Executor, func(data interface{}) interface{} {
				return data
			},
		),
	)(m.Main)(context.Background(), "execute")
	if r != nil {
		if !strings.EqualFold(err.Error(), r.Error()) {
			t.Log("error should be same as predeclared")
			t.FailNow()
		}
	}

	if r == nil {
		t.Log("error shouldn't be nil")
		t.FailNow()
	}
}

func TestLoopArrayWithErrorAndIgnore(t *testing.T) {
	m := NewMock()
	_, r := endpoint.Chain(
		loop_array.Middleware(
			m.Executor, func(data interface{}) interface{} {
				return data
			},
			option.RUN_WITH_ERROR,
		),
	)(m.Main)(context.Background(), "execute")
	if r != nil {
		t.Log("error should be nil")
		t.FailNow()
	}

}
