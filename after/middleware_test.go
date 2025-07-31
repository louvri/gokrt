package after_test

import (
	"context"
	"errors"
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/go-kit/kit/endpoint"
	"github.com/louvri/gokrt/after"
	RUN_WITH_OPTION "github.com/louvri/gokrt/option"
	"github.com/louvri/gosl"
)

type Mock interface {
<<<<<<< HEAD
	Main(ctx context.Context, request any) (any, error)
	First(ctx context.Context, request any) (any, error)
	Second(ctx context.Context, request any) (any, error)
	Third(ctx context.Context, request any) (any, error)
	Error(ctx context.Context, request any) (any, error)
	Insert(ctx context.Context, request any) (any, error)
=======
	Main(ctx context.Context, request any) (any, error)
	First(ctx context.Context, request any) (any, error)
	Second(ctx context.Context, request any) (any, error)
	Third(ctx context.Context, request any) (any, error)
	Error(ctx context.Context, request any) (any, error)
>>>>>>> main
}

var EXPECTED_RESULT string = "main endpoint"
var ErrFoo = errors.New("it's error")

type TestKey int

var TKey TestKey = 13

type mock struct {
	logger *log.Logger
}

func NewMock() Mock {
	return &mock{
		logger: log.Default(),
	}
}

func (m *mock) Main(ctx context.Context, request any) (any, error) {
	return EXPECTED_RESULT, nil
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
	return nil, ErrFoo
}

func (m *mock) Insert(ctx context.Context, request any) (any, error) {
	tobeInsert := request.(string)
	var queryable *gosl.Queryable
	ictx, ok := ctx.Value(gosl.INTERNAL_CONTEXT).(*gosl.InternalContext)
	if ok {
		queryable = ictx.Get(gosl.SQL_KEY).(*gosl.Queryable)
	} else {
		ref := ctx.Value(gosl.SQL_KEY)
		if ref == nil {
			err := errors.New("database is not initialized")
			return nil, err
		}
		queryable = ref.(*gosl.Queryable)
	}
	_, err := queryable.ExecContext(ctx, fmt.Sprintf("INSERT INTO `hello_1` (data) VALUES('%s')", tobeInsert))
	if err != nil {
		return nil, err
	}
	_, err = queryable.ExecContext(ctx, fmt.Sprintf("INSERT INTO `hello_2` (data) VALUES('%s')", tobeInsert))
	if err != nil {
		return nil, err
	}
	return "ok", nil
}
func TestHappyCaseAlter(t *testing.T) {
	m := NewMock()
	ctx := context.WithValue(context.Background(), "key1", "val1")
	ctx = context.WithValue(ctx, "key6", "val1")
	ctx = context.WithValue(ctx, "key7", "val1")
	ctx = context.WithValue(ctx, "key8", "val1")
	ctx = context.WithValue(ctx, "key9", "val1")
	ctx = context.WithValue(ctx, "key10", "val1")
	resp, _ := endpoint.Chain(
		after.Middleware(m.First, func(data any, err error) any {
			t.Log(data)
			ctx = context.WithValue(context.Background(), "key1", data)
			return data
		}, nil),
		after.Middleware(m.Second, func(data any, err error) any {
			t.Log(data)
			ctx = context.WithValue(context.Background(), "key2", data)
			return data
		}, nil),
		after.Middleware(m.Third, func(data any, err error) any {
			t.Log(data)
			ctx = context.WithValue(context.Background(), "key3", data)
			return data
		}, nil),
	)(m.Main)(ctx, "")
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
		after.Middleware(m.First, func(data any, err error) any {
			t.Log(data)
			return data
		}, nil),
		after.Middleware(m.Second, func(data any, err error) any {
			t.Log(err)
			return err
		}, nil),
		after.Middleware(m.Error, func(data any, err error) any {
			t.Log(data)
			return data
		}, nil, RUN_WITH_OPTION.RUN_WITH_ERROR),
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
		after.Middleware(m.First, func(data any, err error) any {
			t.Log(data)
			return data
		}, nil),
		after.Middleware(m.Second, func(data any, err error) any {
			t.Log(err)
			return err
		}, nil),
		after.Middleware(m.Error, func(data any, err error) any {
			t.Log(data)
			return data
		}, nil),
	)(m.Error)(context.Background(), "")
	if result, ok := resp.(string); ok {
		if result != ErrFoo.Error() {
			t.Logf("got '%s' expected '%s'", ErrFoo.Error(), result)
			t.FailNow()
		}
	}
}

func TestAfterWithGosl(t *testing.T) {
	m := NewMock()
	ctx := context.WithValue(context.Background(),
		gosl.SQL_KEY,
		gosl.NewQueryable(gosl.ConnectToDB(
			"root",
			"abcd",
			"localhost",
			"3306",
			"testTx",
			1,
			1,
			2*time.Minute,
			2*time.Minute,
		)))

	resp, _ := endpoint.Chain(
		after.Middleware(m.Insert, func(data any, err error) any {
			var res string
			if tmp, ok := data.(string); ok {
				res = fmt.Sprintf("%s + data 1", tmp)
			}
			return res
		}, nil, RUN_WITH_OPTION.RUN_ASYNC_WAIT),
		after.Middleware(m.Insert, func(data any, err error) any {
			var res string
			if tmp, ok := data.(string); ok {
				res = fmt.Sprintf("%s + data 2", tmp)
			}
			return res
		}, nil, RUN_WITH_OPTION.RUN_ASYNC_WAIT),
		after.Middleware(m.Insert, func(data any, err error) any {
			var res string
			if tmp, ok := data.(string); ok {
				res = fmt.Sprintf("%s + data 3", tmp)
			}
			return res
		}, nil, RUN_WITH_OPTION.RUN_ASYNC_WAIT),
	)(m.Main)(ctx, "")
	if result, ok := resp.(string); ok {
		if result != EXPECTED_RESULT {
			t.Logf("got '%s' expected '%s'", EXPECTED_RESULT, result)
			t.FailNow()
		}
	}
}
