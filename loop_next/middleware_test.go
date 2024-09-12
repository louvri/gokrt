package loop_next_test

import (
	"context"
	"errors"
	"fmt"
	"log"
	"testing"

	"github.com/go-kit/kit/endpoint"
	"github.com/louvri/gokrt/loop_next"
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

var batch = []interface{}{
	"1stIndex",
	"2ndIndex",
	err,
	"3rdIndex",
	"4thIndex",
	err,
	"5thIndex",
	"6thIndex",
	err,
	"7thIndex",
	"8thIndex",
	err,
	"9thIndex",
	"10thIndex",
	err,
}

func NewMock() Mock {
	return &mock{
		logger: log.Default(),
	}
}

var counter = 0

func (m *mock) Main(ctx context.Context, request interface{}) (interface{}, error) {

	if counter >= len(batch) {
		return nil, nil
	}
	fmt.Println(batch[counter])
	if err, ok := batch[counter].(error); ok {
		return nil, err
	}
	return batch[counter], nil
}

func (m *mock) Executor(ctx context.Context, request interface{}) (interface{}, error) {
	m.logger.Printf("output exector endpoint: %s", request)
	if err, ok := request.(error); ok && err != nil {
		return nil, err
	}
	return request, nil
}

func TestLoopNext(t *testing.T) {
	m := NewMock()
	_, err := endpoint.Chain(
		loop_next.Middleware(func(prev, curr interface{}) bool {
			counter += 1
			comparator := len(batch) < counter
			return comparator
		}, func(req, next interface{}) interface{} {
			return counter
		}),
	)(m.Main)(context.Background(), counter)
	if err == nil {
		t.Log("should error since it has to stopped when error")
		t.FailNow()
	}
}

func TestLoopNextNotIgnoreError(t *testing.T) {
	m := NewMock()
	_, err := endpoint.Chain(
		loop_next.Middleware(func(prev, curr interface{}) bool {
			counter += 1
			comparator := len(batch) <= counter
			return comparator
		}, func(req, next interface{}) interface{} {
			return counter
		}, option.RUN_WITH_ERROR),
	)(m.Main)(context.Background(), counter)
	if err == nil {
		t.Log("should error")
		t.FailNow()
	}
}
