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
