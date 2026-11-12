package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go-digital-library-fiber/api/middlewares"
	"go-digital-library-fiber/db/models"

	"github.com/gofiber/fiber/v3"
)

func TestNewAppRegistersPublicAuthRoutes(t *testing.T) {
	t.Setenv("APP_MODE", "production")
	config := middlewares.JWTConfig{SecretKey: "test-secret", ExpiresDuration: 10}
	app := NewApp(nil, nil, config)

	for _, path := range []string{"/api/v1/auth/register", "/api/v1/auth/login"} {
		t.Run(path, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{}`))
			request.Header.Set("Content-Type", "application/json")
			response, err := app.Test(request)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer response.Body.Close()

			if response.StatusCode != http.StatusUnprocessableEntity {
				t.Fatalf("expected public route to validate the request with status %d, got %d", http.StatusUnprocessableEntity, response.StatusCode)
			}
		})
	}
}

func TestNewAppRegistersProtectedRoutes(t *testing.T) {
	t.Setenv("APP_MODE", "production")
	config := middlewares.JWTConfig{SecretKey: "test-secret", ExpiresDuration: 10}
	app := NewApp(nil, nil, config)

	routes := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/profile"},
		{http.MethodPatch, "/api/v1/profile/edit"},
		{http.MethodGet, "/api/v1/categories"},
		{http.MethodGet, "/api/v1/categories/7"},
		{http.MethodPost, "/api/v1/categories"},
		{http.MethodPatch, "/api/v1/categories/7"},
		{http.MethodDelete, "/api/v1/categories/7"},
		{http.MethodGet, "/api/v1/books"},
		{http.MethodGet, "/api/v1/books/category/7"},
		{http.MethodGet, "/api/v1/books/7"},
		{http.MethodPost, "/api/v1/books"},
		{http.MethodPatch, "/api/v1/books/7"},
		{http.MethodDelete, "/api/v1/books/7"},
		{http.MethodGet, "/api/v1/rents"},
		{http.MethodGet, "/api/v1/rents/me"},
		{http.MethodGet, "/api/v1/rents/7"},
		{http.MethodPost, "/api/v1/rents"},
		{http.MethodPatch, "/api/v1/rents/7"},
		{http.MethodDelete, "/api/v1/rents/7"},
	}

	for _, route := range routes {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			response, err := app.Test(httptest.NewRequest(route.method, route.path, nil))
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer response.Body.Close()

			if response.StatusCode != http.StatusUnauthorized {
				t.Fatalf("expected registered route to reject missing token with status %d, got %d", http.StatusUnauthorized, response.StatusCode)
			}
			var body struct {
				Message string `json:"message"`
			}
			if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if body.Message != "Invalid or expired JWT" {
				t.Errorf("unexpected authentication error message %q", body.Message)
			}
		})
	}
}

func TestNewAppAdminRoutesRejectMember(t *testing.T) {
	t.Setenv("APP_MODE", "production")
	config := middlewares.JWTConfig{SecretKey: "test-secret", ExpiresDuration: 10}
	token, err := config.GenerateToken(42, models.Enduser)
	if err != nil {
		t.Fatalf("generate member token: %v", err)
	}
	app := NewApp(nil, nil, config)

	routes := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/api/v1/categories"},
		{http.MethodPatch, "/api/v1/categories/7"},
		{http.MethodDelete, "/api/v1/categories/7"},
		{http.MethodPost, "/api/v1/books"},
		{http.MethodPatch, "/api/v1/books/7"},
		{http.MethodDelete, "/api/v1/books/7"},
		{http.MethodGet, "/api/v1/rents"},
		{http.MethodPatch, "/api/v1/rents/7"},
		{http.MethodDelete, "/api/v1/rents/7"},
	}

	for _, route := range routes {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			request := httptest.NewRequest(route.method, route.path, nil)
			request.Header.Set("Authorization", "Bearer "+token)
			response, err := app.Test(request)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer response.Body.Close()

			if response.StatusCode != fiber.StatusForbidden {
				t.Fatalf("expected member to be forbidden with status %d, got %d", fiber.StatusForbidden, response.StatusCode)
			}
		})
	}
}
