package pgsql

import (
	"context"

	"github.com/emibotz/chat-server/internal/user"
	"github.com/jackc/pgx/v5/pgxpool"
)

type pgsqlDB struct {
	pool *pgxpool.Pool

	users user.Store
}

func New(ctx context.Context, connString string) (*pgsqlDB, error) {
	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		return nil, err
	}

	return &pgsqlDB{
		pool: pool,

		users: &users{pool},
	}, nil
}

func (db *pgsqlDB) Users() user.Store {
	return db.users
}
