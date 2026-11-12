package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go-digital-library-fiber/api/middlewares"
	"go-digital-library-fiber/db/models"
	"go-digital-library-fiber/pkg/rajaongkir"
	"go-digital-library-fiber/pkg/utils"
	"go-digital-library-fiber/rents"
	"go-digital-library-fiber/users"

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
	getErr     error
	listErr    error
	createErr  error
	updateErr  error
	deleteErr  error
	getID      uint
}

type rentUserServiceStub struct {
	profile users.User
	err     error
}

func (s *rentUserServiceStub) GetProfile(context.Context, uint) (users.User, error) {
	return s.profile, s.err
}

func (s *rentUserServiceStub) Update(context.Context, *users.EditProfileRequest, uint) (users.User, error) {
	return s.profile, s.err
}

type deliveryFeeStub struct {
	fee float64
	err error
	req rajaongkir.GetFeeRequest
}

func (s *deliveryFeeStub) GetDeliveryFee(request rajaongkir.GetFeeRequest) (float64, error) {
	s.req = request
	return s.fee, s.err
}

func (s *rentServiceStub) GetAll(_ context.Context, pagination utils.Pagination) ([]rents.Rent, error) {
	s.pagination = pagination
	return s.rents, s.listErr
}

func (s *rentServiceStub) GetByUser(_ context.Context, userID uint) ([]rents.Rent, error) {
	s.userID = userID
	return s.rents, s.listErr
}

func (s *rentServiceStub) GetByID(_ context.Context, id uint) (rents.Rent, error) {
	s.getID = id
	return s.rent, s.getErr
}

func (s *rentServiceStub) Create(_ context.Context, request *rents.RentRequest) error {
	s.request = request
	return s.createErr
}

func (s *rentServiceStub) Update(_ context.Context, request *rents.UpdateRentRequest, id uint) (rents.Rent, error) {
	s.updateReq = request
	s.updatedID = id
	return s.rent, s.updateErr
}

func (s *rentServiceStub) Delete(_ context.Context, id uint) error {
	s.deletedID = id
	return s.deleteErr
}

func TestRentsGetAllPassesPagination(t *testing.T) {
	service := &rentServiceStub{rents: []rents.Rent{{ID: 1}}}
	handler := NewRents(service, &rentUserServiceStub{}, &deliveryFeeStub{})
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
	handler := NewRents(service, &rentUserServiceStub{}, &deliveryFeeStub{})
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
	handler := NewRents(service, &rentUserServiceStub{profile: users.User{DistrictID: 3894}}, &deliveryFeeStub{fee: 125})
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
	handler := NewRents(service, &rentUserServiceStub{}, &deliveryFeeStub{})
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
	handler := NewRents(service, &rentUserServiceStub{}, &deliveryFeeStub{})
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
	handler := NewRents(service, &rentUserServiceStub{}, &deliveryFeeStub{})
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
	handler := NewRents(service, &rentUserServiceStub{}, &deliveryFeeStub{})
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

func TestRentsGetAllReportsServiceFailure(t *testing.T) {
	service := &rentServiceStub{listErr: errors.New("database down")}
	handler := NewRents(service, &rentUserServiceStub{}, &deliveryFeeStub{})
	app, token := protectedTestApp(t, models.Admin, func(app *fiber.App, _ middlewares.JWTConfig) {
		app.Get("/rents", middlewares.VerifyAdmin(), handler.GetAll)
	})
	request := httptestRequest(http.MethodGet, "/rents", "")
	request.Header.Set("Authorization", "Bearer "+token)

	response, err := app.Test(request)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer response.Body.Close()

	assertResponse(t, response, http.StatusInternalServerError, "fetch rents failed")
}

func TestRentsGetByUserReportsServiceFailure(t *testing.T) {
	service := &rentServiceStub{listErr: errors.New("database down")}
	handler := NewRents(service, &rentUserServiceStub{}, &deliveryFeeStub{})
	app, token := protectedTestApp(t, models.Enduser, func(app *fiber.App, _ middlewares.JWTConfig) {
		app.Get("/rents/me", handler.GetByUser)
	})
	request := httptestRequest(http.MethodGet, "/rents/me", "")
	request.Header.Set("Authorization", "Bearer "+token)

	response, err := app.Test(request)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer response.Body.Close()

	assertResponse(t, response, http.StatusInternalServerError, "error occurred")
}

func TestRentsGetByIDCases(t *testing.T) {
	for _, tt := range []struct {
		name       string
		path       string
		role       models.Role
		rent       rents.Rent
		getErr     error
		wantStatus int
		message    string
		wantID     uint
	}{
		{name: "owner reads rent", path: "/rents/1", role: models.Enduser, rent: rents.Rent{ID: 1, UserID: 42}, wantStatus: http.StatusOK, message: "rent found", wantID: 1},
		{name: "admin reads another user's rent", path: "/rents/1", role: models.Admin, rent: rents.Rent{ID: 1, UserID: 7}, wantStatus: http.StatusOK, message: "rent found", wantID: 1},
		{name: "rejects malformed ID", path: "/rents/nope", role: models.Enduser, wantStatus: http.StatusBadRequest, message: "invalid id"},
		{name: "reports lookup failure", path: "/rents/1", role: models.Enduser, getErr: errors.New("database down"), wantStatus: http.StatusInternalServerError, message: "error occurred", wantID: 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			service := &rentServiceStub{rent: tt.rent, getErr: tt.getErr}
			handler := NewRents(service, &rentUserServiceStub{}, &deliveryFeeStub{})
			app, token := protectedTestApp(t, tt.role, func(app *fiber.App, _ middlewares.JWTConfig) {
				app.Get("/rents/:id", handler.GetByID)
			})
			request := httptestRequest(http.MethodGet, tt.path, "")
			request.Header.Set("Authorization", "Bearer "+token)
			response, err := app.Test(request)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer response.Body.Close()

			assertResponse(t, response, tt.wantStatus, tt.message)
			if service.getID != tt.wantID {
				t.Errorf("expected service ID %d, got %d", tt.wantID, service.getID)
			}
		})
	}
}

func TestRentsCreateCases(t *testing.T) {
	for _, tt := range []struct {
		name       string
		body       string
		userErr    error
		feeErr     error
		createErr  error
		wantStatus int
		message    string
	}{
		{name: "creates rent", body: `{"books":[7],"duration":3,"courier":"jne"}`, wantStatus: http.StatusCreated, message: "rent created"},
		{name: "rejects invalid fields", body: `{"books":[],"duration":0,"courier":"unknown"}`, wantStatus: http.StatusUnprocessableEntity, message: "validation failed"},
		{name: "reports user lookup failure", body: `{"books":[7],"duration":3,"courier":"jne"}`, userErr: errors.New("profile lookup failed"), wantStatus: http.StatusInternalServerError, message: "error occurred"},
		{name: "reports delivery fee failure", body: `{"books":[7],"duration":3,"courier":"jne"}`, feeErr: errors.New("fee unavailable"), wantStatus: http.StatusInternalServerError, message: "fee unavailable"},
		{name: "reports rent creation failure", body: `{"books":[7],"duration":3,"courier":"jne"}`, createErr: errors.New("database down"), wantStatus: http.StatusInternalServerError, message: "error occurred"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			service := &rentServiceStub{createErr: tt.createErr}
			userService := &rentUserServiceStub{profile: users.User{DistrictID: 3894}, err: tt.userErr}
			feeService := &deliveryFeeStub{fee: 125, err: tt.feeErr}
			handler := NewRents(service, userService, feeService)
			app, token := protectedTestApp(t, models.Enduser, func(app *fiber.App, _ middlewares.JWTConfig) {
				app.Post("/rents", middlewares.ValidateBody(&rents.RentRequest{}), handler.Create)
			})
			request := jsonRequest(http.MethodPost, "/rents", tt.body)
			request.Header.Set("Authorization", "Bearer "+token)
			response, err := app.Test(request)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer response.Body.Close()

			assertResponse(t, response, tt.wantStatus, tt.message)
			if tt.name == "creates rent" {
				if service.request == nil || service.request.UserID != 42 || service.request.Fee != 125 {
					t.Errorf("rent did not use authenticated user and calculated fee: %+v", service.request)
				}
				if service.request != nil && service.request.ReturnTime.IsZero() {
					t.Error("expected a return date on the created rent request")
				}
				if feeService.req.Destination != "3894" || feeService.req.Courier != "jne" {
					t.Errorf("unexpected delivery fee request: %+v", feeService.req)
				}
			}
		})
	}
}

func TestRentsUpdateCases(t *testing.T) {
	for _, tt := range []struct {
		name       string
		path       string
		body       string
		role       models.Role
		updateErr  error
		wantStatus int
		message    string
		wantID     uint
	}{
		{name: "admin updates rent", path: "/rents/8", body: `{"status":"returned"}`, role: models.Admin, wantStatus: http.StatusOK, message: "rent updated", wantID: 8},
		{name: "rejects malformed ID", path: "/rents/nope", body: `{"status":"returned"}`, role: models.Admin, wantStatus: http.StatusBadRequest, message: "invalid id"},
		{name: "rejects missing status", path: "/rents/8", body: `{}`, role: models.Admin, wantStatus: http.StatusUnprocessableEntity, message: "validation failed"},
		{name: "reports service failure", path: "/rents/8", body: `{"status":"returned"}`, role: models.Admin, updateErr: errors.New("database down"), wantStatus: http.StatusInternalServerError, message: "error occurred", wantID: 8},
		{name: "rejects member", path: "/rents/8", body: `{"status":"returned"}`, role: models.Enduser, wantStatus: http.StatusForbidden, message: "access denied"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			service := &rentServiceStub{rent: rents.Rent{ID: 8}, updateErr: tt.updateErr}
			handler := NewRents(service, &rentUserServiceStub{}, &deliveryFeeStub{})
			app, token := protectedTestApp(t, tt.role, func(app *fiber.App, _ middlewares.JWTConfig) {
				app.Patch("/rents/:id", middlewares.VerifyAdmin(), middlewares.ValidateBody(&rents.UpdateRentRequest{}), handler.Update)
			})
			request := jsonRequest(http.MethodPatch, tt.path, tt.body)
			request.Header.Set("Authorization", "Bearer "+token)
			response, err := app.Test(request)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer response.Body.Close()

			assertResponse(t, response, tt.wantStatus, tt.message)
			if service.updatedID != tt.wantID {
				t.Errorf("expected service ID %d, got %d", tt.wantID, service.updatedID)
			}
		})
	}
}

func TestRentsDeleteCases(t *testing.T) {
	for _, tt := range []struct {
		name       string
		path       string
		role       models.Role
		deleteErr  error
		wantStatus int
		message    string
		wantID     uint
	}{
		{name: "admin deletes rent", path: "/rents/9", role: models.Admin, wantStatus: http.StatusOK, message: "rent deleted", wantID: 9},
		{name: "rejects malformed ID", path: "/rents/nope", role: models.Admin, wantStatus: http.StatusBadRequest, message: "invalid id"},
		{name: "reports delete failure", path: "/rents/9", role: models.Admin, deleteErr: errors.New("database down"), wantStatus: http.StatusInternalServerError, message: "error occurred", wantID: 9},
		{name: "rejects member", path: "/rents/9", role: models.Enduser, wantStatus: http.StatusForbidden, message: "access denied"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			service := &rentServiceStub{deleteErr: tt.deleteErr}
			handler := NewRents(service, &rentUserServiceStub{}, &deliveryFeeStub{})
			app, token := protectedTestApp(t, tt.role, func(app *fiber.App, _ middlewares.JWTConfig) {
				app.Delete("/rents/:id", middlewares.VerifyAdmin(), handler.Delete)
			})
			request := httptestRequest(http.MethodDelete, tt.path, "")
			request.Header.Set("Authorization", "Bearer "+token)
			response, err := app.Test(request)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer response.Body.Close()

			assertResponse(t, response, tt.wantStatus, tt.message)
			if service.deletedID != tt.wantID {
				t.Errorf("expected service ID %d, got %d", tt.wantID, service.deletedID)
			}
		})
	}
}
