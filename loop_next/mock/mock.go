package mock

import (
	"context"
	"errors"
	"log"
)

var Err = errors.New("error appear")
var instance *mock

var Batch = []interface{}{
	"1stIndex",
	"2ndIndex",
	// Err,
	"3rdIndex",
	"4thIndex",
	// Err,
	"5thIndex",
	"6thIndex",
	// Err,
	"7thIndex",
	"8thIndex",
	// Err,
	"9thIndex",
	"10thIndex",
	// Err,
}

type Mock interface {
	Main(ctx context.Context, request interface{}) (interface{}, error)
	Executor(ctx context.Context, request interface{}) (interface{}, error)
	GetCounter() int
	Increment(int)
}

type mock struct {
	logger  *log.Logger
	counter int
}

func NewMock() Mock {
	if instance == nil {
		instance = &mock{
			logger:  log.Default(),
			counter: 0,
		}
	}
	return instance
}

func (m *mock) Main(ctx context.Context, request interface{}) (interface{}, error) {

	current := m.counter
	if current >= len(Batch) {
		return nil, nil
	}
	if err, ok := Batch[current].(error); ok {
		return nil, err
	}
	return Batch[current], nil
}

func (m *mock) Executor(ctx context.Context, request interface{}) (interface{}, error) {
	m.logger.Printf("output exector endpoint: %s", request)
	if err, ok := request.(error); ok && err != nil {
		return nil, err
	}
	return request, nil
}

func (m *mock) GetCounter() int {
	return m.counter
}

func (m *mock) Increment(x int) {
	m.counter += x
}
