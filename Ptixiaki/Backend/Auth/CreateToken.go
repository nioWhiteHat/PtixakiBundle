package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v4"
)

func (a *Auth) CreateToken(uid string, isAdmin bool) (string, error) {
	claims := jwt.MapClaims{
		"uid":   uid,
		"admin": isAdmin,
		"exp":   time.Now().Add(time.Hour * 24).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	secretKey := []byte("your_secret_key")

	tokenString, err := token.SignedString(secretKey)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}