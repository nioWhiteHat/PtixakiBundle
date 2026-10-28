package database

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)
var connectionString = "postgresql://postgres:Ilias.sp.00@spyroiliasdev.ddns.net:5432/eiv-test?sslmode=disable"

func InitDB1() (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(context.Background(), connectionString)
	if err != nil {
		return nil, err
	}

	// Ping the database to ensure the connection string is valid 
	// and the database is reachable
	err = pool.Ping(context.Background())
	if err != nil {
		return nil, err
	}

	return pool, nil
}