package handlers

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"go-digital-library-fiber/api/middlewares"
	"go-digital-library-fiber/categories"
	"go-digital-library-fiber/db/models"
	"go-digital-library-fiber/pkg/utils"

	"github.com/gofiber/fiber/v3"
)

type categoryServiceStub struct {
	categories categories.Category
	list       utils.Pagination
	categoryID uint
	request    *categories.CategoryRequest
	err        error
}

func (s *categoryServiceStub) GetAll(_ context.Context, pagination utils.Pagination) (utils.Pagination, error) {
	s.list = pagination
	return utils.Pagination{Rows: []categories.Category{s.categories}}, s.err
}

func (s *categoryServiceStub) GetByID(_ context.Context, id uint) (categories.Category, error) {
	s.categoryID = id
	return s.categories, s.err
}

func (s *categoryServiceStub) Create(_ context.Context, request *categories.CategoryRequest) (categories.Category, error) {
	s.request = request
	return s.categories, s.err
}

func (s *categoryServiceStub) Update(_ context.Context, request *categories.CategoryRequest, id uint) (categories.Category, error) {
	s.categoryID = id
	s.request = request
	return s.categories, s.err
}

func (s *categoryServiceStub) Delete(_ context.Context, id uint) error {
	s.categoryID = id
	return s.err
}

func TestCategoriesGetAllCases(t *testing.T) {
	for _, tt := range []struct {
		name       string
		err        error
		wantStatus int
		message    string
	}{
		{name: "returns categories", wantStatus: http.StatusOK, message: "all categories"},
		{name: "reports service failure", err: errors.New("db down"), wantStatus: http.StatusInternalServerError, message: "fetch category failed"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			service := &categoryServiceStub{categories: categories.Category{ID: 7, Name: "Fiction"}, err: tt.err}
			handler := NewCategories(service)
			app, token := protectedTestApp(t, models.Enduser, func(app *fiber.App, _ middlewares.JWTConfig) {
				app.Get("/categories", handler.GetAll)
			})

			request := httptestRequest(http.MethodGet, "/categories?page=2&limit=5&sort=name+asc&search=fic", "")
			request.Header.Set("Authorization", "Bearer "+token)
			response, err := app.Test(request)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer response.Body.Close()

			assertResponse(t, response, tt.wantStatus, tt.message)
			if tt.err == nil && (service.list.Page != 2 || service.list.Limit != 5 || service.list.Sort != "name asc" || service.list.Search != "fic" || service.list.Keyword != "name") {
				t.Errorf("unexpected list filters: %+v", service.list)
			}
		})
	}
}

func TestCategoriesGetByIDCases(t *testing.T) {
	for _, tt := range []struct {
		name       string
		path       string
		err        error
		wantStatus int
		message    string
		wantID     uint
	}{
		{name: "returns category", path: "/categories/7", wantStatus: http.StatusOK, message: "category found", wantID: 7},
		{name: "rejects malformed ID", path: "/categories/nope", wantStatus: http.StatusBadRequest, message: "invalid id"},
		{name: "reports service failure", path: "/categories/7", err: errors.New("database down"), wantStatus: http.StatusInternalServerError, message: "error occurred", wantID: 7},
	} {
		t.Run(tt.name, func(t *testing.T) {
			service := &categoryServiceStub{categories: categories.Category{ID: 7, Name: "Fiction"}, err: tt.err}
			handler := NewCategories(service)
			app, token := protectedTestApp(t, models.Enduser, func(app *fiber.App, _ middlewares.JWTConfig) {
				app.Get("/categories/:id", handler.GetByID)
			})
			request := httptestRequest(http.MethodGet, tt.path, "")
			request.Header.Set("Authorization", "Bearer "+token)
			response, err := app.Test(request)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer response.Body.Close()

			assertResponse(t, response, tt.wantStatus, tt.message)
			if service.categoryID != tt.wantID {
				t.Errorf("expected service ID %d, got %d", tt.wantID, service.categoryID)
			}
		})
	}
}

func TestCategoriesCreateCases(t *testing.T) {
	for _, tt := range []struct {
		name       string
		body       string
		err        error
		wantStatus int
		message    string
	}{
		{name: "creates category", body: `{"name":"Fiction","description":"Fiction books"}`, wantStatus: http.StatusCreated, message: "category created"},
		{name: "rejects missing field", body: `{"name":"Fiction"}`, wantStatus: http.StatusUnprocessableEntity, message: "validation failed"},
		{name: "reports service failure", body: `{"name":"Fiction","description":"Fiction books"}`, err: errors.New("db down"), wantStatus: http.StatusInternalServerError, message: "error occurred"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			service := &categoryServiceStub{categories: categories.Category{ID: 7, Name: "Fiction"}, err: tt.err}
			handler := NewCategories(service)
			app, token := protectedTestApp(t, models.Admin, func(app *fiber.App, _ middlewares.JWTConfig) {
				app.Post("/categories", middlewares.VerifyAdmin(), middlewares.ValidateBody(&categories.CategoryRequest{}), handler.Create)
			})
			request := jsonRequest(http.MethodPost, "/categories", tt.body)
			request.Header.Set("Authorization", "Bearer "+token)
			response, err := app.Test(request)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer response.Body.Close()

			assertResponse(t, response, tt.wantStatus, tt.message)
		})
	}
}

func TestCategoriesUpdateCases(t *testing.T) {
	for _, tt := range []struct {
		name       string
		path       string
		body       string
		err        error
		wantStatus int
		message    string
		wantID     uint
	}{
		{name: "updates category", path: "/categories/7", body: `{"name":"New name","description":"Updated description"}`, wantStatus: http.StatusOK, message: "category updated", wantID: 7},
		{name: "rejects malformed ID", path: "/categories/nope", body: `{"name":"New name","description":"Updated description"}`, wantStatus: http.StatusBadRequest, message: "invalid id"},
		{name: "rejects invalid body", path: "/categories/7", body: `{"name":"New name"}`, wantStatus: http.StatusUnprocessableEntity, message: "validation failed"},
		{name: "reports service failure", path: "/categories/7", body: `{"name":"New name","description":"Updated description"}`, err: errors.New("db down"), wantStatus: http.StatusInternalServerError, message: "error occurred", wantID: 7},
	} {
		t.Run(tt.name, func(t *testing.T) {
			service := &categoryServiceStub{categories: categories.Category{ID: 7}, err: tt.err}
			handler := NewCategories(service)
			app, token := protectedTestApp(t, models.Admin, func(app *fiber.App, _ middlewares.JWTConfig) {
				app.Patch("/categories/:id", middlewares.VerifyAdmin(), middlewares.ValidateBody(&categories.CategoryRequest{}), handler.Update)
			})
			request := jsonRequest(http.MethodPatch, tt.path, tt.body)
			request.Header.Set("Authorization", "Bearer "+token)
			response, err := app.Test(request)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer response.Body.Close()

			assertResponse(t, response, tt.wantStatus, tt.message)
			if service.categoryID != tt.wantID {
				t.Errorf("expected service ID %d, got %d", tt.wantID, service.categoryID)
			}
		})
	}
}

func TestCategoriesDeleteCases(t *testing.T) {
	for _, tt := range []struct {
		name       string
		path       string
		err        error
		wantStatus int
		message    string
		wantID     uint
	}{
		{name: "deletes category", path: "/categories/7", wantStatus: http.StatusOK, message: "category deleted", wantID: 7},
		{name: "rejects malformed ID", path: "/categories/nope", wantStatus: http.StatusBadRequest, message: "invalid id"},
		{name: "reports service failure", path: "/categories/7", err: errors.New("db down"), wantStatus: http.StatusInternalServerError, message: "error occurred", wantID: 7},
	} {
		t.Run(tt.name, func(t *testing.T) {
			service := &categoryServiceStub{err: tt.err}
			handler := NewCategories(service)
			app, token := protectedTestApp(t, models.Admin, func(app *fiber.App, _ middlewares.JWTConfig) {
				app.Delete("/categories/:id", middlewares.VerifyAdmin(), handler.Delete)
			})
			request := httptestRequest(http.MethodDelete, tt.path, "")
			request.Header.Set("Authorization", "Bearer "+token)
			response, err := app.Test(request)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer response.Body.Close()

			assertResponse(t, response, tt.wantStatus, tt.message)
			if service.categoryID != tt.wantID {
				t.Errorf("expected service ID %d, got %d", tt.wantID, service.categoryID)
			}
		})
	}
}

func TestAdminCategoryRouteRejectsMember(t *testing.T) {
	handler := NewCategories(&categoryServiceStub{})
	app, token := protectedTestApp(t, models.Enduser, func(app *fiber.App, _ middlewares.JWTConfig) {
		app.Post("/categories", middlewares.VerifyAdmin(), middlewares.ValidateBody(&categories.CategoryRequest{}), handler.Create)
	})
	request := jsonRequest(http.MethodPost, "/categories", `{"name":"Fiction","description":"Fiction books"}`)
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
