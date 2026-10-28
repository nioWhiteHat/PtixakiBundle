package auth

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Auth struct {
	Db *pgxpool.Pool
	Ctx context.Context
}

