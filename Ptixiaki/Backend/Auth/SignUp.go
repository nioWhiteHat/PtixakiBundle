package auth

import (
	"errors"
	"net"
	"regexp"
	"strings"

	"github.com/oklog/ulid/v2"
	"golang.org/x/crypto/bcrypt"
)

func (a *Auth) CreateUser(email string, password string, admin bool) (string, bool, error) {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return "", false, errors.New("invalid email format")
	}

	domain := parts[1]
	mx, err := net.LookupMX(domain)
	if err != nil || len(mx) == 0 {
		return "", false, errors.New("email domain cannot receive mail")
	}

	if len(password) <= 8 {
		return "", false, errors.New("password must be longer than 8 characters")
	}

	isValid, _ := regexp.MatchString(`^[a-zA-Z0-9]+$`, password)
	if !isValid {
		return "", false, errors.New("password must contain only english letters and numbers")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", false, err
	}

	uid := GetNewUserUlid()
	err = a.Db.QueryRow(a.Ctx, "INSERT INTO user_credentials (uid, email, password, admin) VALUES ($1, $2, $3, $4) RETURNING uid", uid, email, string(hashedPassword), admin).Scan(&uid)
	if err != nil {
		return "", false, err
	}

	return uid, true, nil
}
func GetNewUserUlid() string {
	ulid := "user_" + ulid.Make().String()
	return ulid
}