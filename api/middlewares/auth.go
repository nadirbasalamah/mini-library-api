package middlewares

import (
	"errors"
	"go-digital-library-fiber/db/models"
	"net/http"
	"time"

	jwtware "github.com/gofiber/contrib/v3/jwt"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/extractors"
	"github.com/golang-jwt/jwt/v5"
)

type JWTCustomClaims struct {
	ID   int         `json:"id"`
	Role models.Role `json:"role"`
	jwt.RegisteredClaims
}

type JWTConfig struct {
	SecretKey       string
	ExpiresDuration int
}

type contextKey string

const userContextKey = contextKey("user")

func (jwtCfg *JWTConfig) Init() fiber.Handler {
	return jwtware.New(jwtware.Config{
		SigningKey:   jwtware.SigningKey{Key: []byte(jwtCfg.SecretKey)},
		ErrorHandler: jwtError,
		Claims:       &JWTCustomClaims{},
		Extractor:    extractors.FromAuthHeader("Bearer"),
	})
}

func jwtError(c fiber.Ctx, err error) error {
	if err.Error() == "Missing or malformed JWT" {
		return c.Status(fiber.StatusBadRequest).
			JSON(fiber.Map{"status": "error", "message": "Missing or malformed JWT", "data": nil})
	}
	return c.Status(fiber.StatusUnauthorized).
		JSON(fiber.Map{"status": "error", "message": "Invalid or expired JWT", "data": nil})
}

func (jwtCfg *JWTConfig) GenerateToken(userID int, role models.Role) (string, error) {
	expire := jwt.NewNumericDate(time.Now().Local().Add(time.Minute * time.Duration(int64(jwtCfg.ExpiresDuration))))

	claims := &JWTCustomClaims{
		ID:        userID,
		ExpiresAt: expire,
		Role:      role,
	}

	rawToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	token, err := rawToken.SignedString([]byte(jwtCfg.SecretKey))

	if err != nil {
		return "", err
	}

	return token, nil
}

func GetUser(c fiber.Ctx) (*JWTCustomClaims, error) {
	token := jwtware.FromContext(c)
	if token == nil {
		return nil, errors.New("invalid token")
	}

	claims, ok := token.Claims.(*JWTCustomClaims)
	if !ok {
		return nil, errors.New("invalid claims")
	}

	return claims, nil
}

func GetUserID(c fiber.Ctx) (int, error) {
	claim, err := GetUser(c)

	if err != nil {
		return 0, errors.New("invalid token")
	}

	return claim.ID, nil
}

func VerifyToken() fiber.Handler {
	return func(c fiber.Ctx) error {
		userData, err := GetUser(c)
		if userData == nil || err != nil {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
				"message": "invalid token",
			})
		}

		c.Locals("userData", userData)

		return c.Next()
	}
}

func VerifyAdmin() fiber.Handler {
	return func(c fiber.Ctx) error {
		user, err := GetUser(c)

		if err != nil {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
				"message": "invalid token",
			})
		}

		if user.Role != models.Admin {
			return c.Status(http.StatusForbidden).JSON(fiber.Map{
				"message": "access denied",
			})
		}

		return c.Next()
	}
}
