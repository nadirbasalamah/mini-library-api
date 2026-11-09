package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go-digital-library-fiber/api/middlewares"
	"go-digital-library-fiber/db/models"
	"go-digital-library-fiber/pkg/utils"
	"go-digital-library-fiber/rents"

	"github.com/gofiber/fiber/v3"
)

type rentServiceStub struct {
	userID     uint
	request    *rents.RentRequest
	pagination utils.Pagination
	rent       rents.Rent
	rents      []rents.Rent
	updatedID  uint
	updateReq  *rents.UpdateRentRequest
	deletedID  uint
}

func (s *rentServiceStub) GetAll(_ context.Context, pagination utils.Pagination) ([]rents.Rent, error) {
	s.pagination = pagination
	return s.rents, nil
}

func (s *rentServiceStub) GetByUser(_ context.Context, userID uint) ([]rents.Rent, error) {
	s.userID = userID
	return s.rents, nil
}

func (s *rentServiceStub) GetByID(context.Context, uint) (rents.Rent, error) {
	return s.rent, nil
}

func (s *rentServiceStub) Create(_ context.Context, request *rents.RentRequest) error {
	s.request = request
	return nil
}

func (s *rentServiceStub) Update(_ context.Context, request *rents.UpdateRentRequest, id uint) (rents.Rent, error) {
	s.updateReq = request
	s.updatedID = id
	return s.rent, nil
}

func (s *rentServiceStub) Delete(_ context.Context, id uint) error {
	s.deletedID = id
	return nil
}

func TestRentsGetAllPassesPagination(t *testing.T) {
	service := &rentServiceStub{rents: []rents.Rent{{ID: 1}}}
	handler := NewRents(service)
	app := fiber.New()
	app.Get("/rents", handler.GetAll)

	request := httptest.NewRequest(http.MethodGet, "/rents?page=2&limit=5&sort=id+desc&search=jne", nil)
	response, err := app.Test(request)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.StatusCode)
	}
	if service.pagination.Page != 2 || service.pagination.Limit != 5 {
		t.Errorf("unexpected pagination: %+v", service.pagination)
	}
	if service.pagination.Sort != "id desc" || service.pagination.Search != "jne" || service.pagination.Keyword != "courier" {
		t.Errorf("unexpected rent filters: %+v", service.pagination)
	}
}

func TestRentsGetByUserUsesAuthenticatedUserID(t *testing.T) {
	service := &rentServiceStub{rents: []rents.Rent{{ID: 1, UserID: 42}}}
	handler := NewRents(service)
	app, token := rentTestApp(t, handler, func(app *fiber.App) {
		app.Get("/rents/me", handler.GetByUser)
	})

	request := httptest.NewRequest(http.MethodGet, "/rents/me", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	response, err := app.Test(request)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.StatusCode)
	}
	if service.userID != 42 {
		t.Errorf("expected service to receive user ID 42, got %d", service.userID)
	}

	var body struct {
		Status string       `json:"status"`
		Data   []rents.Rent `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if body.Status != "success" || len(body.Data) != 1 || body.Data[0].ID != 1 {
		t.Errorf("unexpected response: %+v", body)
	}
}

func TestRentsCreateUsesAuthenticatedUserID(t *testing.T) {
	service := &rentServiceStub{}
	handler := NewRents(service)
	app, token := rentTestApp(t, handler, func(app *fiber.App) {
		app.Post("/rents", func(ctx fiber.Ctx) error {
			ctx.Locals("validatedBody", &rents.RentRequest{
				Books:    []uint{7},
				Duration: 3,
				Courier:  "jne",
				UserID:   99,
			})
			return ctx.Next()
		}, handler.Create)
	})

	request := httptest.NewRequest(http.MethodPost, "/rents", strings.NewReader("{}"))
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", "application/json")
	response, err := app.Test(request)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, response.StatusCode)
	}
	if service.request == nil || service.request.UserID != 42 {
		t.Fatalf("expected created rent to use authenticated user ID 42, got %+v", service.request)
	}
}

func TestRentsGetByIDRejectsAnotherUsersRent(t *testing.T) {
	service := &rentServiceStub{rent: rents.Rent{ID: 1, UserID: 7}}
	handler := NewRents(service)
	app, token := rentTestApp(t, handler, func(app *fiber.App) {
		app.Get("/rents/:id", handler.GetByID)
	})

	request := httptest.NewRequest(http.MethodGet, "/rents/1", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	response, err := app.Test(request)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, response.StatusCode)
	}
}

func TestRentsGetByIDAllowsOwner(t *testing.T) {
	service := &rentServiceStub{rent: rents.Rent{ID: 1, UserID: 42}}
	handler := NewRents(service)
	app, token := rentTestApp(t, handler, func(app *fiber.App) {
		app.Get("/rents/:id", handler.GetByID)
	})

	request := httptest.NewRequest(http.MethodGet, "/rents/1", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	response, err := app.Test(request)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.StatusCode)
	}
}

func TestRentsUpdatePassesIDAndRequest(t *testing.T) {
	service := &rentServiceStub{rent: rents.Rent{ID: 8}}
	handler := NewRents(service)
	app := fiber.New()
	app.Patch("/rents/:id", func(ctx fiber.Ctx) error {
		ctx.Locals("validatedBody", &rents.UpdateRentRequest{Status: "returned"})
		return ctx.Next()
	}, handler.Update)

	response, err := app.Test(httptest.NewRequest(http.MethodPatch, "/rents/8", nil))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.StatusCode)
	}
	if service.updatedID != 8 || service.updateReq == nil || service.updateReq.Status != "returned" {
		t.Errorf("unexpected update arguments: ID=%d request=%+v", service.updatedID, service.updateReq)
	}
}

func TestRentsDeletePassesID(t *testing.T) {
	service := &rentServiceStub{}
	handler := NewRents(service)
	app := fiber.New()
	app.Delete("/rents/:id", handler.Delete)

	response, err := app.Test(httptest.NewRequest(http.MethodDelete, "/rents/9", nil))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.StatusCode)
	}
	if service.deletedID != 9 {
		t.Errorf("expected rent 9 to be deleted, got %d", service.deletedID)
	}
}

func rentTestApp(t *testing.T, handler Rents, registerRoutes func(*fiber.App)) (*fiber.App, string) {
	t.Helper()

	config := middlewares.JWTConfig{SecretKey: "test-secret", ExpiresDuration: 10}
	token, err := config.GenerateToken(42, models.Enduser)
	if err != nil {
		t.Fatalf("failed to generate test token: %v", err)
	}

	app := fiber.New()
	app.Use(config.Init(), middlewares.VerifyToken())
	registerRoutes(app)

	return app, token
}
