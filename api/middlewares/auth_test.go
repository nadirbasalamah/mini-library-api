package middlewares

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-digital-library-fiber/db/models"

	"github.com/gofiber/fiber/v3"
)

func TestJWTMiddlewareCases(t *testing.T) {
	tests := []struct {
		name       string
		token      string
		tokenRole  models.Role
		expiration int
		wantStatus int
		message    string
	}{
		{name: "missing token", wantStatus: http.StatusUnauthorized, message: "Invalid or expired JWT"},
		{name: "invalid token", token: "not-a-jwt", wantStatus: http.StatusUnauthorized, message: "Invalid or expired JWT"},
		{name: "expired token", tokenRole: models.Enduser, expiration: -1, wantStatus: http.StatusUnauthorized, message: "Invalid or expired JWT"},
		{name: "valid token", tokenRole: models.Enduser, expiration: 10, wantStatus: http.StatusNoContent},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := JWTConfig{SecretKey: "test-secret", ExpiresDuration: tt.expiration}
			app := fiber.New()
			app.Get("/protected", config.Init(), VerifyToken(), func(ctx fiber.Ctx) error {
				return ctx.SendStatus(http.StatusNoContent)
			})

			request := httptest.NewRequest(http.MethodGet, "/protected", nil)
			if tt.tokenRole != "" {
				token, err := config.GenerateToken(42, tt.tokenRole)
				if err != nil {
					t.Fatalf("generate token: %v", err)
				}
				request.Header.Set("Authorization", "Bearer "+token)
			} else if tt.token != "" {
				request.Header.Set("Authorization", "Bearer "+tt.token)
			}

			response, err := app.Test(request)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer response.Body.Close()

			if response.StatusCode != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, response.StatusCode)
			}
			if tt.message != "" {
				var body struct {
					Message string `json:"message"`
				}
				if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
					t.Fatalf("decode response: %v", err)
				}
				if body.Message != tt.message {
					t.Errorf("expected message %q, got %q", tt.message, body.Message)
				}
			}
		})
	}
}

func TestVerifyAdminCases(t *testing.T) {
	for _, tt := range []struct {
		name       string
		role       models.Role
		wantStatus int
	}{
		{name: "allows admin", role: models.Admin, wantStatus: http.StatusNoContent},
		{name: "rejects member", role: models.Enduser, wantStatus: http.StatusForbidden},
	} {
		t.Run(tt.name, func(t *testing.T) {
			config := JWTConfig{SecretKey: "test-secret", ExpiresDuration: 10}
			token, err := config.GenerateToken(42, tt.role)
			if err != nil {
				t.Fatalf("generate token: %v", err)
			}
			app := fiber.New()
			app.Get("/admin", config.Init(), VerifyToken(), VerifyAdmin(), func(ctx fiber.Ctx) error {
				return ctx.SendStatus(http.StatusNoContent)
			})
			request := httptest.NewRequest(http.MethodGet, "/admin", nil)
			request.Header.Set("Authorization", "Bearer "+token)

			response, err := app.Test(request)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer response.Body.Close()

			if response.StatusCode != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, response.StatusCode)
			}
		})
	}
}
