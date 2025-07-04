package alter_test

import (
	"context"
	"errors"
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/go-kit/kit/endpoint"
	"github.com/louvri/gokrt/after"
	"github.com/louvri/gokrt/alter"
	RUN_WITH_OPTION "github.com/louvri/gokrt/option"
	"github.com/louvri/gosl"
)

type TestKey int

var TKey TestKey = 13

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

func TestBeforeRun(t *testing.T) {
	m := NewMock()
	resp, _ := endpoint.Chain(
		alter.Middleware(m.First, func(data interface{}, err error) interface{} {
			t.Log(data)
			return data
		}, func(original, data interface{}, err error) (interface{}, error) {
			t.Log(original)
			return original, nil
		}, RUN_WITH_OPTION.EXECUTE_BEFORE),
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
		if result != "second endpoint" {
			t.Logf("got %s but it should be 'second endpoint'", result)
			t.FailNow()
		}
	}
}

func TestAlterAfter(t *testing.T) {
	m := NewMock()
	_, err := endpoint.Chain(
		after.Middleware(m.First, func(data interface{}, err error) interface{} {
			return data
		}, nil),
		alter.Middleware(m.Main, func(data interface{}, err error) interface{} {
			t.Log(data)
			return data
		}, func(original, data interface{}, err error) (interface{}, error) {
			t.Log(data)
			return data, nil
		}),
	)(m.Error)(context.Background(), "")
	if err == nil {
		t.Log("should error")
		t.FailNow()
	}
}

func TestWithMultipleValueContext(t *testing.T) {
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
	ctx = context.WithValue(ctx,
		TKey,
		gosl.NewQueryable(gosl.ConnectToDB(
			"root",
			"abcd",
			"localhost",
			"3306",
			"testTx2",
			1,
			1,
			2*time.Minute,
			2*time.Minute,
		)))
	resp, err := endpoint.Chain(
		alter.Middleware(m.First, func(data interface{}, err error) interface{} {
			return data
		}, func(original, data interface{}, err error) (interface{}, error) {
			return data, err
		}),
		alter.Middleware(ep, func(data interface{}, err error) interface{} {
			return data
		}, func(original, data interface{}, err error) (interface{}, error) {
			return data, err
		}),
	)(m.Main)(ctx, "request1")
	if err != nil {
		t.Log(err.Error())
		t.FailNow()
	}
	if result, ok := resp.(string); ok {
		if result != "first endpoint" {
			t.Logf("got '%s' expected 'first endpoint'", result)
			t.FailNow()
		}
	}
}

func ep(ctx context.Context, req interface{}) (interface{}, error) {
	ctx, kit := gosl.New(ctx)

	_, ok := ctx.Value(gosl.SQL_KEY).(*gosl.Queryable)
	if !ok {
		return nil, errors.New("sql not initiated")
	}

	err := kit.RunInTransaction(
		ctx,
		func(ctx context.Context) error {
			err := svc1(ctx)
			if err != nil {
				return err
			}

			err = svc2(ctx)
			if err != nil {
				return err
			}
			return err
		},
	)
	return "done ep", err
}

func svc1(ctx context.Context) error {
	ctx, kit := gosl.New(ctx)
	err := kit.RunInTransaction(ctx, func(ctx context.Context) error {
		if err := kit.ContextSwitch(ctx, TKey); err == nil {
			err = repo(ctx, "SWITCHsatutigabelas1")
			if err != nil {
				return err
			}
		}
		if err := kit.ContextReset(ctx); err == nil {
			err = repo(ctx, "RESETsatutigabelas1")
			if err != nil {
				return err
			}
		}
		return nil
	})
	return err
}

func repo(ctx context.Context, data string) error {
	ctx, kit := gosl.New(ctx)
	err := kit.RunInTransaction(ctx, func(ctx context.Context) error {
		var queryable *gosl.Queryable
		ictx, ok := ctx.Value(gosl.INTERNAL_CONTEXT).(*gosl.InternalContext)
		if ok {
			queryable = ictx.Get(gosl.SQL_KEY).(*gosl.Queryable)
		} else {
			ref := ctx.Value(gosl.SQL_KEY)
			if ref == nil {
				err := errors.New("database is not initialized")
				return err
			}
			queryable = ref.(*gosl.Queryable)
		}
		_, err := queryable.ExecContext(ctx, fmt.Sprintf("INSERT INTO `hello_1` (data) VALUES('%s')", data))
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

func svc2(ctx context.Context) error {
	ctx, kit := gosl.New(ctx)
	err := kit.RunInTransaction(ctx, func(ctx context.Context) error {
		if err := kit.ContextSwitch(ctx, TKey); err == nil {
			err = repo(ctx, "SWITCHsatutigabelas2")
			if err != nil {
				return nil
			}

		}
		if err := kit.ContextReset(ctx); err == nil {
			err = repo(ctx, "RESETsatutigabelas2")
			if err != nil {
				return err
			}
		}
		return nil
	})
	return err
}

func reset(ctx context.Context) error {
	ctx, kit := gosl.New(ctx)
	var queryable *gosl.Queryable
	queryable, ok := ctx.Value(gosl.SQL_KEY).(*gosl.Queryable)
	if !ok {
		return errors.New("sql not initiated")
	}
	_, err := queryable.ExecContext(ctx, "DELETE FROM `hello_1`")
	if err != nil {
		return err
	}
	_, err = queryable.ExecContext(ctx, "DELETE FROM `hello_2`")
	if err != nil {
		return err
	}

	if err = kit.ContextSwitch(ctx, TKey); err == nil {
		var queryable *gosl.Queryable

		ictx, ok := ctx.Value(gosl.INTERNAL_CONTEXT).(*gosl.InternalContext)
		if ok {
			queryable = ictx.Get(gosl.SQL_KEY).(*gosl.Queryable)
			ctx = ictx.Base()
		} else {
			ref := ctx.Value(gosl.SQL_KEY)
			if ref == nil {
				err = errors.New("database is not initialized")
				return err
			}
			queryable = ref.(*gosl.Queryable)
		}

		_, err := queryable.ExecContext(ctx, "DELETE FROM `hello_1`")
		if err != nil {
			return err
		}
		_, err = queryable.ExecContext(ctx, "DELETE FROM `hello_2`")
		if err != nil {
			return err
		}
	}
	return nil
}
