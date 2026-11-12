package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-digital-library-fiber/api/middlewares"
	"go-digital-library-fiber/auth"
	"go-digital-library-fiber/db/models"

	"github.com/gofiber/fiber/v3"
)

type authServiceStub struct {
	user        auth.User
	registerErr error
	loginErr    error
}

func (s *authServiceStub) Register(context.Context, *auth.RegisterRequest) (auth.User, error) {
	return s.user, s.registerErr
}

func (s *authServiceStub) Login(context.Context, *auth.LoginRequest) (auth.User, error) {
	return s.user, s.loginErr
}

func TestAuthRegisterCases(t *testing.T) {
	tests := []struct {
		name        string
		body        string
		registerErr error
		wantStatus  int
		wantMessage string
	}{
		{
			name:        "valid registration",
			body:        `{"username":"reader1","email":"reader1@example.com","password":"StrongPass1!","address":"1 Main St","province_id":1,"city_id":2,"district_id":3}`,
			wantStatus:  http.StatusCreated,
			wantMessage: "user registered",
		},
		{
			name:        "malformed JSON",
			body:        `{"username":`,
			wantStatus:  http.StatusBadRequest,
			wantMessage: "invalid request",
		},
		{
			name:        "invalid request fields",
			body:        `{"username":"","email":"bad-email","password":"weak","address":"","province_id":0,"city_id":0,"district_id":0}`,
			wantStatus:  http.StatusUnprocessableEntity,
			wantMessage: "validation failed",
		},
		{
			name:        "service failure",
			body:        `{"username":"reader1","email":"reader1@example.com","password":"StrongPass1!","address":"1 Main St","province_id":1,"city_id":2,"district_id":3}`,
			registerErr: errors.New("database unavailable"),
			wantStatus:  http.StatusInternalServerError,
			wantMessage: "error occurred",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &authServiceStub{user: auth.User{ID: 1, Username: "reader1", Email: "reader1@example.com"}, registerErr: tt.registerErr}
			handler := NewAuth(service, middlewares.JWTConfig{SecretKey: "test-secret", ExpiresDuration: 10})
			app := fiber.New()
			app.Post("/register", handler.Register)

			response, err := app.Test(jsonRequest(http.MethodPost, "/register", tt.body))
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer response.Body.Close()

			assertResponse(t, response, tt.wantStatus, tt.wantMessage)
			if tt.name == "valid registration" {
				var body struct {
					Data map[string]json.RawMessage `json:"data"`
				}
				if err := fiberJSONDecode(response, &body); err != nil {
					t.Fatalf("decode registration response: %v", err)
				}
				if len(body.Data) == 0 {
					t.Fatal("expected registered user data")
				}
				if _, exists := body.Data["password"]; exists {
					t.Error("registration response exposed the password")
				}
			}
		})
	}
}

func TestAuthLoginCases(t *testing.T) {
	tests := []struct {
		name        string
		body        string
		loginErr    error
		wantStatus  int
		wantMessage string
	}{
		{
			name:        "valid credentials",
			body:        `{"email":"reader1@example.com","password":"StrongPass1!"}`,
			wantStatus:  http.StatusOK,
			wantMessage: "login success",
		},
		{
			name:        "malformed JSON",
			body:        `{"email":`,
			wantStatus:  http.StatusBadRequest,
			wantMessage: "invalid request",
		},
		{
			name:        "invalid credentials shape",
			body:        `{"email":"not-an-email","password":""}`,
			wantStatus:  http.StatusUnprocessableEntity,
			wantMessage: "validation failed",
		},
		{
			name:        "service rejects credentials",
			body:        `{"email":"reader1@example.com","password":"wrong"}`,
			loginErr:    errors.New("record not found"),
			wantStatus:  http.StatusInternalServerError,
			wantMessage: "error occurred",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &authServiceStub{user: auth.User{ID: 1, Username: "reader1", Role: "user"}, loginErr: tt.loginErr}
			handler := NewAuth(service, middlewares.JWTConfig{SecretKey: "test-secret", ExpiresDuration: 10})
			app := fiber.New()
			app.Post("/login", handler.Login)

			response, err := app.Test(jsonRequest(http.MethodPost, "/login", tt.body))
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer response.Body.Close()

			assertResponse(t, response, tt.wantStatus, tt.wantMessage)
			if tt.name == "valid credentials" {
				var body struct {
					Data string `json:"data"`
				}
				if err := fiberJSONDecode(response, &body); err != nil {
					t.Fatalf("decode login response: %v", err)
				}
				if body.Data == "" {
					t.Fatal("expected login response to contain a token")
				}
			}
		})
	}
}

func TestAuthLoginTokenCanAuthorizeProtectedRoute(t *testing.T) {
	config := middlewares.JWTConfig{SecretKey: "test-secret", ExpiresDuration: 10}
	handler := NewAuth(&authServiceStub{user: auth.User{ID: 42, Role: string(models.Enduser)}}, config)
	app := fiber.New()
	app.Post("/login", handler.Login)
	app.Get("/protected", config.Init(), middlewares.VerifyToken(), func(ctx fiber.Ctx) error {
		return ctx.SendStatus(http.StatusNoContent)
	})

	loginResponse, err := app.Test(jsonRequest(http.MethodPost, "/login", `{"email":"reader1@example.com","password":"StrongPass1!"}`))
	if err != nil {
		t.Fatalf("login request failed: %v", err)
	}
	defer loginResponse.Body.Close()

	var loginBody struct {
		Data string `json:"data"`
	}
	if err := fiberJSONDecode(loginResponse, &loginBody); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	if loginBody.Data == "" {
		t.Fatal("expected login to return a token")
	}

	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	request.Header.Set("Authorization", "Bearer "+loginBody.Data)
	protectedResponse, err := app.Test(request)
	if err != nil {
		t.Fatalf("protected request failed: %v", err)
	}
	defer protectedResponse.Body.Close()

	if protectedResponse.StatusCode != http.StatusNoContent {
		t.Fatalf("expected protected route status %d, got %d", http.StatusNoContent, protectedResponse.StatusCode)
	}
}
