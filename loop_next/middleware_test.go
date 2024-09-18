package loop_next_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/go-kit/kit/endpoint"
	"github.com/jmoiron/sqlx"
	"github.com/louvri/gokrt/after"
	"github.com/louvri/gokrt/loop_next"
	"github.com/louvri/gokrt/loop_next/mock"
	"github.com/louvri/gokrt/option"
	"github.com/louvri/gosl"
)

var ctx context.Context
var tmpIndex = 0

func TestLoopNextNotIgnoreError(t *testing.T) {
	m := mock.NewMock()
	_, err := endpoint.Chain(
		loop_next.Middleware(func(prev, curr interface{}) bool {
			comparator := len(mock.Err.Error()) <= m.GetCounter()
			m.Increment(1)
			return comparator
		}, func(req, next interface{}) interface{} {
			return tmpIndex
		}),
		after.Middleware(m.Executor, func(data interface{}, err error) interface{} {
			if data != nil && err == nil {
				return data
			}
			return err
		}),
	)(m.Main)(context.Background(), tmpIndex)
	if err != nil {
		t.Log(err.Error())
		t.FailNow()
	}
}

func TestLoopNext(t *testing.T) {
	m := mock.NewMock()
	_, err := endpoint.Chain(
		loop_next.Middleware(func(prev, curr interface{}) bool {
			m.Increment(1)
			comparator := len(mock.Batch) <= m.GetCounter()
			return comparator
		}, func(req, next interface{}) interface{} {
			return m.GetCounter()
		}, option.RUN_WITH_ERROR),
		after.Middleware(m.Executor, func(data interface{}, err error) interface{} {
			if data != nil && err == nil {
				return data
			}
			return err
		}, option.RUN_WITH_ERROR),
	)(m.Main)(context.Background(), m.GetCounter())
	if err == nil {
		t.Log("should error")
		t.FailNow()
	}
}

func TestRunTransaction(t *testing.T) {
	ctx = context.Background()
	m := mock.NewMock()
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
	mDB := mock.NewMockDB(db)
	ctx = context.WithValue(ctx, gosl.SQL_KEY, gosl.NewQueryable(db))

	_, err = endpoint.Chain(
		loop_next.Middleware(func(prev, curr interface{}) bool {
			m.Increment(1)
			comparator := len(mock.Batch) <= m.GetCounter()
			return comparator
		}, func(req, next interface{}) interface{} {
			res := m.GetCounter()
			return res
		}, option.RUN_WITH_TRANSACTION, option.RUN_WITH_ERROR),
		after.Middleware(
			mDB.Upsert, func(data interface{}, err error) interface{} {
				if data != nil && err == nil {
					return data
				}
				return err
			}, option.RUN_ASYNC,
		),
	)(m.Main)(ctx, tmpIndex)
	if err != nil {
		t.Log(err.Error())
		t.FailNow()
	}
}
