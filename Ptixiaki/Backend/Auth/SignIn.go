package auth

import (
	"database/sql"
	"errors"

	"golang.org/x/crypto/bcrypt"
)

func (a *Auth) SignIn(email string, password string) (string, bool, error) {
	var uid string
	var hashedPassword string
	var isAdmin bool

	err := a.Db.QueryRow(a.Ctx, "SELECT uid, password, admin FROM user_credentials WHERE email = $1", email).Scan(&uid, &hashedPassword, &isAdmin)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", false, errors.New("invalid email or password")
		}
		return "", false, err
	}

	err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	if err != nil {
		return "", false, errors.New("invalid email or password")
	}

	return uid, isAdmin, nil
}