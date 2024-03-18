package test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/go-kit/kit/endpoint"
	"github.com/louvri/gokrt/after"
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
