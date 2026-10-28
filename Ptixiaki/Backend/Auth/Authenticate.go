package auth

import (
	"context"
	"errors"
	"net/http"

	"github.com/golang-jwt/jwt/v4"
)

func Authenticate(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("token")
		if err != nil {
			http.Error(w, "Unauthorized: No token provided", http.StatusUnauthorized)
			return
		}

		tokenString := cookie.Value
		secretKey := []byte("your_secret_key")

		token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("unexpected signing method")
			}
			return secretKey, nil
		})

		if err != nil {
			http.SetCookie(w, &http.Cookie{
				Name:     "token",
				Value:    "",
				Path:     "/",
				MaxAge:   -1,
				HttpOnly: true,
				Secure:   true,
				SameSite: http.SameSiteStrictMode,
			})

			if errors.Is(err, jwt.ErrTokenExpired) {
				http.Error(w, "Unauthorized: Token expired", http.StatusUnauthorized)
				return
			}

			http.Error(w, "Unauthorized: Malformed or invalid token", http.StatusUnauthorized)
			return
		}

		if !token.Valid {
			http.SetCookie(w, &http.Cookie{
				Name:     "token",
				Value:    "",
				Path:     "/",
				MaxAge:   -1,
				HttpOnly: true,
				Secure:   true,
				SameSite: http.SameSiteStrictMode,
			})
			http.Error(w, "Unauthorized: Invalid token", http.StatusUnauthorized)
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			http.Error(w, "Unauthorized: Invalid token claims", http.StatusUnauthorized)
			return
		}

		uid, ok := claims["uid"].(string)
		if !ok || uid == "" {
			http.Error(w, "Unauthorized: UID missing from token", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), "user_uid", uid)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}