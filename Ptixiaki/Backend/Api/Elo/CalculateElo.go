package elo

import (
	p "Ptixiaki/Worker/Pipelines"
	explain "Ptixiaki/Worker/explainability"
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
)

type Queryer interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

func CalculateElo(ctx context.Context, pool Queryer, uid, transaction string) (float64, float64, error) {
	if err := explain.ValidateTransaction(transaction); err != nil {
		return 0, 0, err
	}
	var geoPremium float64
	var residual *float64
	err := pool.QueryRow(ctx, "SELECT geo_premium, residual_percentage FROM estimation_data WHERE uid = $1", uid).Scan(&geoPremium, &residual)
	if err != nil {
		return 0, 0, fmt.Errorf("read property scoring data: %w", err)
	}
	var geoMean, geoStd float64
	err = pool.QueryRow(ctx, "SELECT mean, std FROM global_stats WHERE stat_name = $1", "Geo_stats_"+transaction).Scan(&geoMean, &geoStd)
	if err != nil {
		return 0, 0, fmt.Errorf("read global GeoPremium statistics: %w", err)
	}
	geoZ := p.CalculateZScore(geoPremium, geoMean, geoStd)
	// A missing residual is not a zero-valued residual. Both deal terms are
	// neutral; no price statistics or deal-comparable queries are needed.
	if residual == nil {
		return p.CalculateEloScore(0, 0, geoZ), geoZ, nil
	}
	var residualMean, residualStd float64
	err = pool.QueryRow(ctx, "SELECT mean, std FROM global_stats WHERE stat_name = $1", "Price_stats_"+transaction).Scan(&residualMean, &residualStd)
	if err != nil {
		return 0, 0, fmt.Errorf("read global residual statistics: %w", err)
	}
	localZ, err := CalculateLocalDealValue(ctx, pool, uid, transaction, *residual)
	if err != nil {
		return 0, 0, err
	}
	return p.CalculateEloScore(p.CalculateZScore(*residual, residualMean, residualStd), localZ, geoZ), geoZ, nil
}
func CalculateLocalDealValue(ctx context.Context, pool Queryer, uid, transaction string, residual float64) (float64, error) {
	uids, err := p.GetData(ctx, pool, transaction, uid)
	if err != nil {
		return 0, err
	}
	rows, err := pool.Query(ctx, "SELECT uid, residual_percentage FROM estimation_data WHERE uid = ANY($1) AND uid != $2 AND residual_percentage IS NOT NULL AND asking_price_per_sqm > 0", uids, uid)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	properties := make([]p.EstimationRow, 0, len(uids))
	for rows.Next() {
		var property p.EstimationRow
		if err := rows.Scan(&property.UID, &property.ResidualPercentage); err != nil {
			return 0, err
		}
		properties = append(properties, property)
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	std, mean := p.CalculateMathForDealValue(properties)
	return p.CalculateZScore(residual, mean, std), nil
}
