package test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/go-kit/kit/endpoint"
	"github.com/louvri/gokrt/after"
	"github.com/louvri/gokrt/alter"
	"github.com/louvri/gokrt/on_eof"
	"github.com/louvri/gokrt/sys_key"
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
