package mock

import (
	"context"
	"fmt"
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/louvri/gosl"
)

type MockDB interface {
	Upsert(context.Context, interface{}) (interface{}, error)
}

type mockDB struct {
	logger *log.Logger
	db     *sqlx.DB
}

var instanceDb *mockDB

func NewMockDB(db *sqlx.DB) MockDB {
	if instanceDb == nil {
		instanceDb = &mockDB{
			logger: log.Default(),
			db:     db,
		}
	}
	return instanceDb
}

func (m *mockDB) Upsert(ctx context.Context, request interface{}) (interface{}, error) {
	var tobeInsert string
	if request == nil {
		return nil, nil
	}
	if tmp, ok := request.(string); ok {
		tobeInsert = tmp
	} else if tmp, ok := request.(error); ok {
		fmt.Printf("found error on upsert: %v \n", tmp)
		return nil, tmp
	}
	var queryable *gosl.Queryable

	if tmp, ok := ctx.Value(gosl.SQL_KEY).(*gosl.Queryable); ok {
		queryable = tmp
	} else {
		ictx, ok := ctx.Value(gosl.INTERNAL_CONTEXT).(*gosl.InternalContext)
		if ok {
			queryable = ictx.Get(gosl.SQL_KEY).(*gosl.Queryable)
			ctx = ictx.Base()
		}
	}
	fmt.Printf("tobe inserted on upsert: %s \n", tobeInsert)
	query := fmt.Sprintf("INSERT INTO hello_1(`data`) VALUES('%s')", tobeInsert)
	res, err := queryable.ExecContext(ctx, query)
	if err != nil {
		return nil, err
	}
	return res.LastInsertId()
}
