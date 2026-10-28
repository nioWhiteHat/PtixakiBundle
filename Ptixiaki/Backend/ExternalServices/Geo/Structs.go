package GetGeoData

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Geo struct {
	Db  *pgxpool.Pool
	Ctx context.Context
}