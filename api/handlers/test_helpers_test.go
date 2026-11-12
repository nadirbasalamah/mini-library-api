package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-digital-library-fiber/api/middlewares"
	"go-digital-library-fiber/db/models"

	"github.com/gofiber/fiber/v3"
)

func assertResponse(t *testing.T, response *http.Response, status int, message string) {
	t.Helper()

	if response.StatusCode != status {
		t.Fatalf("expected status %d, got %d", status, response.StatusCode)
	}

	data, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	var body struct {
		Status  string `json:"status"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(data, &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	response.Body = io.NopCloser(bytes.NewReader(data))
	if body.Message != message {
		t.Errorf("expected message %q, got %q", message, body.Message)
	}
	if status < http.StatusBadRequest && body.Status != "success" {
		t.Errorf("expected success response status, got %q", body.Status)
	}
	if status >= http.StatusBadRequest && body.Status == "success" {
		t.Errorf("expected failure response status, got %q", body.Status)
	}
}

func fiberJSONDecode(response *http.Response, target any) error {
	return json.NewDecoder(response.Body).Decode(target)
}

func jsonRequest(method, path, body string) *http.Request {
	request := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")
	return request
}

func httptestRequest(method, path, body string) *http.Request {
	return httptest.NewRequest(method, path, bytes.NewBufferString(body))
}

func protectedTestApp(t *testing.T, role models.Role, register func(*fiber.App, middlewares.JWTConfig)) (*fiber.App, string) {
	t.Helper()

	config := middlewares.JWTConfig{SecretKey: "test-secret", ExpiresDuration: 10}
	token, err := config.GenerateToken(42, role)
	if err != nil {
		t.Fatalf("generate test token: %v", err)
	}

	app := fiber.New()
	app.Use(config.Init(), middlewares.VerifyToken())
	register(app, config)
	return app, token
}
