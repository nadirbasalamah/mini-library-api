package utils

import (
	"go-digital-library-fiber/db/models"
	"strconv"

	"time"

	"github.com/golang-jwt/jwt/v5"
)

func GenerateJWT(userID int, role models.Role) (string, error) {
	duration, err := strconv.Atoi(GetConfig("JWT_EXPIRE_DURATION"))

	if err != nil {
		return "", err
	}

	claims := jwt.MapClaims{
		"id":   userID,
		"role": role,
		"exp":  time.Now().Add(time.Minute * time.Duration(int64(duration))).Unix(),
	}

	rawToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	token, err := rawToken.SignedString([]byte(GetConfig("JWT_SECRET_KEY")))

	if err != nil {
		return "", err
	}

	return token, nil
}
