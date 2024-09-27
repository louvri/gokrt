package loop_array_test

import (
	"context"
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

func (m *mock) Insert(ctx context.Context, request interface{}) (interface{}, error) {
	var tobeInsert string
	if tmp, ok := request.(string); ok {
		tobeInsert = tmp
	} else if tmp, ok := request.(error); ok {
		fmt.Printf("found error on upsert: %v \n", tmp)
		return nil, tmp
	}
	queryable := ctx.Value(gosl.SQL_KEY).(*gosl.Queryable)
	fmt.Printf("tobe inserted on upsert: %s \n", tobeInsert)
	query := fmt.Sprintf("INSERT INTO trx_table(`values`) VALUES('%s')", tobeInsert)
	res, err := queryable.ExecContext(ctx, query)
	if err != nil {
		return nil, err
	}
	return res.LastInsertId()
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
			},
			option.RUN_WITH_ERROR,
		),
	)(m.Main)(context.Background(), "execute")
	if r != nil {
		t.Log("error should be nil")
		t.FailNow()
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
		}),
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
		tmp := strings.Split(err.Error(), " || ")
		if len(tmp) != 2 {
			t.Log("error should have len 2 since it's injected with 2 errors")
			t.FailNow()
		}
	}
}
