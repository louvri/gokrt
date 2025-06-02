package loop_array_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"testing"

	"github.com/go-kit/kit/endpoint"
	"github.com/jmoiron/sqlx"
	"github.com/louvri/gokrt/loop_array"
	"github.com/louvri/gokrt/option"
	"github.com/louvri/gosl"
)

var err = errors.New("error appear")

type Mock interface {
	Main(ctx context.Context, request interface{}) (interface{}, error)
	Executor(ctx context.Context, request interface{}) (interface{}, error)
	Insert(ctx context.Context, request interface{}) (interface{}, error)
	InsertWithTx(ctx context.Context, request interface{}) (interface{}, error)
}

type mock struct {
	logger *log.Logger
	db     *sqlx.DB
}

func NewMock(db *sqlx.DB) Mock {
	return &mock{
		logger: log.Default(),
		db:     db,
	}
}

func (m *mock) InsertWithTx(ctx context.Context, request interface{}) (interface{}, error) {
	var tobeInsert string
	var cErr error
	if tmp, ok := request.(string); ok {
		tobeInsert = tmp
	} else if tmp, ok := request.(error); ok {
		cErr = tmp
	}
	var result int64
	queryable := ctx.Value(gosl.SQL_KEY).(*gosl.Queryable)
	fmt.Printf("tobe inserted on upsert: %s \n", tobeInsert)
	kit := gosl.New(ctx)

	c, err := kit.RunInTransaction(ctx, func(ctx context.Context) (context.Context, error) {
		query := fmt.Sprintf("INSERT INTO trx_table(`values`) VALUES('%s')", tobeInsert)
		res, err := queryable.ExecContext(ctx, query)
		if cErr != nil {
			err = cErr
		}
		if err != nil {
			return ctx, err
		}
		result, _ = res.LastInsertId()
		return ctx, nil
	})
	return gosl.Context{
		Ctx:  c,
		Data: result,
	}, err

}

func (m *mock) Insert(ctx context.Context, request interface{}) (interface{}, error) {
	var tobeInsert string
	var cErr error
	if tmp, ok := request.(string); ok {
		tobeInsert = tmp
	} else if tmp, ok := request.(error); ok {
		cErr = tmp
	}
	var result int64
	queryable := ctx.Value(gosl.SQL_KEY).(*gosl.Queryable)
	fmt.Printf("tobe inserted on upsert: %s \n", tobeInsert)
	// kit := gosl.New(ctx)

	query := fmt.Sprintf("INSERT INTO trx_table(`values`) VALUES('%s')", tobeInsert)
	res, err := queryable.ExecContext(ctx, query)
	if cErr != nil {
		err = cErr
	}
	result, _ = res.LastInsertId()
	return gosl.Context{
		Ctx:  ctx,
		Data: result,
	}, err
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
	m := NewMock(nil)
	_, r := endpoint.Chain(
		loop_array.Middleware(
			m.Executor, func(data interface{}) interface{} {
				return data
			}, func(original, data interface{}, err error) {
				// no op
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
	m := NewMock(nil)
	_, r := endpoint.Chain(
		loop_array.Middleware(
			m.Executor, func(data interface{}) interface{} {
				return data
			}, func(original, data interface{}, err error) {
				// no op
			},
			option.RUN_WITH_ERROR,
		),
	)(m.Main)(context.Background(), "execute")
	if r != nil {
		decoded := make([]map[string]interface{}, 0)
		if curr := json.Unmarshal([]byte(r.Error()), &decoded); curr != nil {
			t.Log("error should be able decoded to array interface")
			t.FailNow()
		}

		if len(decoded) != 1 {
			t.Log("error have one, as the pre test function declared sum of error")
			t.FailNow()
		}
	}

}

func TestLoopRunInTransaction(t *testing.T) {
	ctx := context.Background()

	db, err := sqlx.Connect("mysql", fmt.Sprintf(
		"%s:%s@(%s:%s)/%s",
		"root",
		"abcd",
		"localhost",
		"3306",
		"testTx"))
	if err != nil {
		t.Log(err.Error())
		t.FailNow()
	}
	m := NewMock(db)

	ctx = context.WithValue(ctx, gosl.SQL_KEY, gosl.NewQueryable(db))
	_, err = endpoint.Chain(
		loop_array.Middleware(m.Insert, func(data interface{}) interface{} {
			return data
		}, func(original, data interface{}, err error) {
			// no op
		}, option.RUN_IN_TRANSACTION),
	)(func(context.Context, interface{}) (interface{}, error) {
		return []interface{}{
			"1stIndex",
			"2ndIndex",
			"3rdIndex",
			"4thIndex",
			"5thIndex",
		}, nil
	})(ctx, "execute")
	if err != nil {
		t.Log(err.Error())
		t.FailNow()
	}
}

func TestLoopRunInTransactionWithError(t *testing.T) {
	ctx := context.Background()

	db, err := sqlx.Connect("mysql", fmt.Sprintf(
		"%s:%s@(%s:%s)/%s",
		"root",
		"abcd",
		"localhost",
		"3306",
		"testTx"))
	if err != nil {
		t.Log(err.Error())
		t.FailNow()
	}
	m := NewMock(db)

	ctx = context.WithValue(ctx, gosl.SQL_KEY, gosl.NewQueryable(db))
	_, err = endpoint.Chain(
		loop_array.Middleware(m.Insert, func(data interface{}) interface{} {
			return data
		}, func(original, data interface{}, err error) {

		}, option.RUN_IN_TRANSACTION),
	)(func(context.Context, interface{}) (interface{}, error) {
		return []interface{}{
			"1stIndex",
			errors.New("first error"),
			"3rdIndex",
			"4thIndex",
			errors.New("second error"),
			"5thIndex",
		}, nil
	})(ctx, "execute")
	if err == nil {
		t.Log("should return error")
		t.FailNow()
	}
}

func TestLoopRunWithError(t *testing.T) {
	ctx := context.Background()

	db, err := sqlx.Connect("mysql", fmt.Sprintf(
		"%s:%s@(%s:%s)/%s",
		"root",
		"abcd",
		"localhost",
		"3306",
		"testTx"))
	if err != nil {
		t.Log(err.Error())
		t.FailNow()
	}
	m := NewMock(db)

	ctx = context.WithValue(ctx, gosl.SQL_KEY, gosl.NewQueryable(db))
	_, err = endpoint.Chain(
		loop_array.Middleware(m.Insert, func(data interface{}) interface{} {
			return data
		}, func(original, data interface{}, err error) {
			// no op
		}, option.RUN_WITH_ERROR),
	)(func(context.Context, interface{}) (interface{}, error) {
		return []interface{}{
			"1stIndex",
			errors.New("first error"),
			"3rdIndex",
			"4thIndex",
			errors.New("second error"),
			"5thIndex",
		}, nil
	})(ctx, "execute")
	if err == nil {
		t.Log("should return error")
		t.FailNow()
	}

	if err != nil {
		decoded := make([]map[string]interface{}, 0)
		if curr := json.Unmarshal([]byte(err.Error()), &decoded); curr != nil {
			t.Log("error should be able decoded to array interface")
			t.FailNow()
		}

		if len(decoded) != 2 {
			t.Log("error have one, as the pre test function declared sum of error")
			t.FailNow()
		}
	}
}

func TestLoopFailedTransact(t *testing.T) {
	m := NewMock(nil)
	_, r := endpoint.Chain(
		loop_array.Middleware(
			m.Executor, func(data interface{}) interface{} {
				return data
			}, func(original, data interface{}, err error) {
				// no op
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

func TestLoopRunInTransactionWithErrorToRollback(t *testing.T) {
	ctx := context.Background()

	db, err := sqlx.Connect("mysql", fmt.Sprintf(
		"%s:%s@(%s:%s)/%s",
		"root",
		"abcd",
		"localhost",
		"3306",
		"testTx"))
	if err != nil {
		t.Log(err.Error())
		t.FailNow()
	}
	m := NewMock(db)

	ctx = context.WithValue(ctx, gosl.SQL_KEY, gosl.NewQueryable(db))

	// Chain the endpoints with the loop_array middleware
	_, err = endpoint.Chain(
		loop_array.Middleware(m.Insert, func(data interface{}) interface{} {
			return data
		}, func(original, data interface{}, err error) {
		}, option.RUN_IN_TRANSACTION),
	)(func(context.Context, interface{}) (interface{}, error) {
		return []interface{}{
			"1stIndex",
			errors.New("first error"), // This will trigger a rollback
			"3rdIndex",
			"4thIndex",
			errors.New("second error"),
			"5thIndex",
		}, nil
	})(ctx, "execute")

	if err == nil {
		t.Log("should return error")
		t.FailNow()
	}
}

func TestNested(t *testing.T) {
	ctx := context.Background()

	db, err := sqlx.Connect("mysql", fmt.Sprintf(
		"%s:%s@(%s:%s)/%s",
		"root",
		"abcd",
		"localhost",
		"3306",
		"testTx"))
	if err != nil {
		t.Log(err.Error())
		t.FailNow()
	}
	m := NewMock(db)
	ctx = context.WithValue(ctx, gosl.SQL_KEY, gosl.NewQueryable(db))
	kit := gosl.New(ctx)

	if _, err := kit.RunInTransaction(ctx, func(ctx context.Context) (context.Context, error) {
		r, err := m.InsertWithTx(ctx, "data1")
		if err != nil {
			if tmp, ok := r.(gosl.Context); ok {
				return tmp.Ctx, err
			}
			return ctx, err
		}

		r, err = m.InsertWithTx(ctx, errors.New("error1"))
		if err != nil {
			if tmp, ok := r.(gosl.Context); ok {
				return tmp.Ctx, err
			}
			return ctx, err
		}
		return ctx, nil
	}); err == nil {
		t.Log("should be error")
		t.FailNow()
	}
}

func TestLoopRunInTransactionWithNoError(t *testing.T) {
	ctx := context.Background()

	db, err := sqlx.Connect("mysql", fmt.Sprintf(
		"%s:%s@(%s:%s)/%s",
		"root",
		"abcd",
		"localhost",
		"3306",
		"testTx"))
	if err != nil {
		t.Log(err.Error())
		t.FailNow()
	}
	m := NewMock(db)

	ctx = context.WithValue(ctx, gosl.SQL_KEY, gosl.NewQueryable(db))
	// Chain the endpoints with the loop_array middleware
	_, err = endpoint.Chain(
		loop_array.Middleware(m.Insert, func(data interface{}) interface{} {
			return data
		}, func(original, data interface{}, err error) {
		}, option.RUN_IN_TRANSACTION),
	)(func(context.Context, interface{}) (interface{}, error) {
		return []interface{}{
			"1stIndex",
			"3rdIndex",
			"4thIndex",
			"5thIndex",
		}, nil
	})(ctx, "execute")

	if err == nil {
		t.Log("should return error")
		t.FailNow()
	}
}
