package UserActions

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

func (u *User) RemovePropertyFromPortfolio(db *pgxpool.Pool, ctx context.Context, propertyUID string) (bool, error) {
	result, err := db.Exec(ctx, `
		DELETE FROM user_properties
		WHERE user_uid = $1 AND property_uid = $2
	`, u.Uid, propertyUID)
	if err != nil {
		return false, fmt.Errorf("delete property from user portfolio: %w", err)
	}
	return result.RowsAffected() == 1, nil
}
