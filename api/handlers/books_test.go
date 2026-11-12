package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-digital-library-fiber/api/middlewares"
	"go-digital-library-fiber/books"
	"go-digital-library-fiber/db/models"
	"go-digital-library-fiber/pkg/fileupload"
	"go-digital-library-fiber/pkg/utils"

	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"github.com/gofiber/fiber/v3"
)

type bookServiceStub struct {
	categoryPagination utils.Pagination
	categoryID         uint
	book               books.Book
	bookID             uint
	pagination         utils.Pagination
	createdBook        *books.BookRequest
	updatedBook        *books.BookRequest
	updatedID          uint
	deletedID          uint
	listErr            error
	getErr             error
	categoryErr        error
	createErr          error
	updateErr          error
	deleteErr          error
}

func (s *bookServiceStub) GetAll(_ context.Context, pagination utils.Pagination) (utils.Pagination, error) {
	s.pagination = pagination
	return utils.Pagination{Rows: []books.Book{{ID: 1, Title: "Dune"}}}, s.listErr
}

func (s *bookServiceStub) GetByID(_ context.Context, id uint) (books.Book, error) {
	s.bookID = id
	return s.book, s.getErr
}

func (s *bookServiceStub) GetByCategory(_ context.Context, pagination utils.Pagination, categoryID uint) (utils.Pagination, error) {
	s.categoryPagination = pagination
	s.categoryID = categoryID
	return utils.Pagination{Rows: []books.Book{{ID: 1, Title: "Dune"}}}, s.categoryErr
}

func (s *bookServiceStub) Create(_ context.Context, request *books.BookRequest) (books.Book, error) {
	s.createdBook = request
	return s.book, s.createErr
}

func (s *bookServiceStub) Update(_ context.Context, request *books.BookRequest, id uint) (books.Book, error) {
	s.updatedBook = request
	s.updatedID = id
	return s.book, s.updateErr
}

func (s *bookServiceStub) Delete(_ context.Context, id uint) error {
	s.deletedID = id
	return s.deleteErr
}

type uploaderStub struct {
	uploadResult *fileupload.FileUploadResult
	uploaded     bool
	uploadErr    error
	deletedIDs   []string
	deleteErr    error
}

func (s *uploaderStub) UploadFile(context.Context, *multipart.FileHeader, uploader.UploadParams) (*fileupload.FileUploadResult, error) {
	s.uploaded = true
	return s.uploadResult, s.uploadErr
}

func (s *uploaderStub) DeleteFile(_ context.Context, fileID, _ string, _ uploader.UploadParams) (*uploader.DestroyResult, error) {
	s.deletedIDs = append(s.deletedIDs, fileID)
	return &uploader.DestroyResult{}, s.deleteErr
}

func TestBooksGetByCategory(t *testing.T) {
	service := &bookServiceStub{}
	handler := NewBooks(service, &uploaderStub{})
	app := fiber.New()
	app.Get("/books/category/:id", handler.GetByCategory)

	request := httptest.NewRequest(http.MethodGet, "/books/category/7?page=2&limit=5&sort=title+asc&search=Du", nil)
	response, err := app.Test(request)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.StatusCode)
	}

	if service.categoryID != 7 {
		t.Errorf("expected category ID 7, got %d", service.categoryID)
	}
	if service.categoryPagination.Page != 2 || service.categoryPagination.Limit != 5 {
		t.Errorf("unexpected pagination: %+v", service.categoryPagination)
	}
	if service.categoryPagination.Sort != "title asc" || service.categoryPagination.Search != "Du" || service.categoryPagination.Keyword != "title" {
		t.Errorf("unexpected book filters: %+v", service.categoryPagination)
	}

	var body struct {
		Status  string `json:"status"`
		Message string `json:"message"`
		Data    struct {
			Rows []books.Book `json:"rows"`
		} `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if body.Status != "success" || body.Message != "books by category" {
		t.Errorf("unexpected response: %+v", body)
	}
	if len(body.Data.Rows) != 1 || body.Data.Rows[0].Title != "Dune" {
		t.Errorf("unexpected book rows: %+v", body.Data.Rows)
	}
}

func TestBooksUpdateReplacesImage(t *testing.T) {
	service := &bookServiceStub{book: books.Book{ID: 8, ImageID: "previous-image"}}
	upload := &uploaderStub{uploadResult: &fileupload.FileUploadResult{
		SecureURL: "https://example.com/new.png",
		PublicID:  "new-image",
	}}
	handler := NewBooks(service, upload)
	app := fiber.New()
	app.Patch("/books/:id", func(ctx fiber.Ctx) error {
		ctx.Locals("validatedBody", &books.BookRequest{Title: "Updated"})
		return ctx.Next()
	}, handler.Update)

	body, contentType := multipartRequestWithFile(t, "cover.png")
	request := httptest.NewRequest(http.MethodPatch, "/books/8", body)
	request.Header.Set("Content-Type", contentType)
	response, err := app.Test(request)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.StatusCode)
	}
	if !upload.uploaded {
		t.Fatal("expected replacement image to be uploaded")
	}
	if len(upload.deletedIDs) != 1 || upload.deletedIDs[0] != "previous-image" {
		t.Errorf("expected previous image to be deleted, got %v", upload.deletedIDs)
	}
	if service.updatedID != 8 || service.updatedBook == nil {
		t.Fatalf("expected book 8 to be updated, got ID %d and request %+v", service.updatedID, service.updatedBook)
	}
	if service.updatedBook.ImageLink != "https://example.com/new.png" || service.updatedBook.ImageID != "new-image" {
		t.Errorf("updated request has wrong image: %+v", service.updatedBook)
	}
}

func TestBooksUpdateWithoutFileKeepsCurrentImage(t *testing.T) {
	service := &bookServiceStub{book: books.Book{ID: 8, ImageLink: "https://example.com/current.png", ImageID: "current-image"}}
	upload := &uploaderStub{}
	handler := NewBooks(service, upload)
	app := fiber.New()
	app.Patch("/books/:id", func(ctx fiber.Ctx) error {
		ctx.Locals("validatedBody", &books.BookRequest{Title: "Updated"})
		return ctx.Next()
	}, handler.Update)

	response, err := app.Test(httptest.NewRequest(http.MethodPatch, "/books/8", nil))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.StatusCode)
	}
	if service.updatedBook == nil || service.updatedBook.ImageLink != "https://example.com/current.png" || service.updatedBook.ImageID != "current-image" {
		t.Errorf("update did not preserve existing image: %+v", service.updatedBook)
	}
	if upload.uploaded || len(upload.deletedIDs) != 0 {
		t.Errorf("unexpected image operations: uploaded=%t deleted=%v", upload.uploaded, upload.deletedIDs)
	}
}

func TestBooksDeleteRemovesImage(t *testing.T) {
	service := &bookServiceStub{book: books.Book{ID: 9, ImageID: "book-image"}}
	upload := &uploaderStub{}
	handler := NewBooks(service, upload)
	app := fiber.New()
	app.Delete("/books/:id", handler.Delete)

	response, err := app.Test(httptest.NewRequest(http.MethodDelete, "/books/9", nil))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.StatusCode)
	}
	if len(upload.deletedIDs) != 1 || upload.deletedIDs[0] != "book-image" {
		t.Errorf("expected book image to be deleted, got %v", upload.deletedIDs)
	}
	if service.deletedID != 9 {
		t.Errorf("expected book 9 to be deleted, got %d", service.deletedID)
	}
}

func TestBooksDeleteStopsWhenImageDeletionFails(t *testing.T) {
	service := &bookServiceStub{book: books.Book{ID: 9, ImageID: "book-image"}}
	upload := &uploaderStub{deleteErr: errors.New("cloudinary unavailable")}
	handler := NewBooks(service, upload)
	app := fiber.New()
	app.Delete("/books/:id", handler.Delete)

	response, err := app.Test(httptest.NewRequest(http.MethodDelete, "/books/9", nil))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, response.StatusCode)
	}
	if service.deletedID != 0 {
		t.Errorf("book was deleted despite image deletion failure: %d", service.deletedID)
	}
}

func multipartRequestWithFile(t *testing.T, filename string) (*bytes.Buffer, string) {
	t.Helper()

	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	file, err := writer.CreateFormFile("file", filename)
	if err != nil {
		t.Fatalf("failed to create form file: %v", err)
	}
	if _, err := file.Write([]byte("image data")); err != nil {
		t.Fatalf("failed to write form file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("failed to close multipart writer: %v", err)
	}

	return body, writer.FormDataContentType()
}

func TestBooksGetAllCases(t *testing.T) {
	for _, tt := range []struct {
		name       string
		err        error
		wantStatus int
		message    string
	}{
		{name: "returns books", wantStatus: http.StatusOK, message: "all books"},
		{name: "reports service failure", err: errors.New("database down"), wantStatus: http.StatusInternalServerError, message: "fetch books failed"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			service := &bookServiceStub{listErr: tt.err}
			handler := NewBooks(service, &uploaderStub{})
			app, token := protectedTestApp(t, models.Enduser, func(app *fiber.App, _ middlewares.JWTConfig) {
				app.Get("/books", handler.GetAll)
			})
			request := httptestRequest(http.MethodGet, "/books?page=3&limit=4&sort=title+asc&search=dune", "")
			request.Header.Set("Authorization", "Bearer "+token)
			response, err := app.Test(request)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer response.Body.Close()

			assertResponse(t, response, tt.wantStatus, tt.message)
			if tt.err == nil && (service.pagination.Page != 3 || service.pagination.Limit != 4 || service.pagination.Sort != "title asc" || service.pagination.Search != "dune" || service.pagination.Keyword != "title") {
				t.Errorf("unexpected pagination: %+v", service.pagination)
			}
		})
	}
}

func TestBooksGetByCategoryFailureCases(t *testing.T) {
	for _, tt := range []struct {
		name       string
		path       string
		err        error
		wantStatus int
		message    string
	}{
		{name: "rejects malformed category ID", path: "/books/category/nope", wantStatus: http.StatusBadRequest, message: "invalid id"},
		{name: "reports category service failure", path: "/books/category/7", err: errors.New("database down"), wantStatus: http.StatusInternalServerError, message: "error occurred"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			service := &bookServiceStub{categoryErr: tt.err}
			handler := NewBooks(service, &uploaderStub{})
			app, token := protectedTestApp(t, models.Enduser, func(app *fiber.App, _ middlewares.JWTConfig) {
				app.Get("/books/category/:id", handler.GetByCategory)
			})
			request := httptestRequest(http.MethodGet, tt.path, "")
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

func TestBooksGetByIDCases(t *testing.T) {
	for _, tt := range []struct {
		name       string
		path       string
		err        error
		wantStatus int
		message    string
		wantID     uint
	}{
		{name: "returns book", path: "/books/12", wantStatus: http.StatusOK, message: "book found", wantID: 12},
		{name: "rejects malformed ID", path: "/books/nope", wantStatus: http.StatusBadRequest, message: "invalid id"},
		{name: "reports lookup failure", path: "/books/12", err: errors.New("database down"), wantStatus: http.StatusInternalServerError, message: "error occurred", wantID: 12},
	} {
		t.Run(tt.name, func(t *testing.T) {
			service := &bookServiceStub{book: books.Book{ID: 12, Title: "Dune"}, getErr: tt.err}
			handler := NewBooks(service, &uploaderStub{})
			app, token := protectedTestApp(t, models.Enduser, func(app *fiber.App, _ middlewares.JWTConfig) {
				app.Get("/books/:id", handler.GetByID)
			})
			request := httptestRequest(http.MethodGet, tt.path, "")
			request.Header.Set("Authorization", "Bearer "+token)
			response, err := app.Test(request)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer response.Body.Close()

			assertResponse(t, response, tt.wantStatus, tt.message)
			if service.bookID != tt.wantID {
				t.Errorf("expected service ID %d, got %d", tt.wantID, service.bookID)
			}
		})
	}
}

func TestBooksCreateCases(t *testing.T) {
	for _, tt := range []struct {
		name       string
		filename   string
		fields     bool
		uploadErr  error
		createErr  error
		wantStatus int
		message    string
	}{
		{name: "creates book", filename: "cover.png", fields: true, wantStatus: http.StatusCreated, message: "book created"},
		{name: "requires image file", fields: true, wantStatus: http.StatusUnprocessableEntity, message: "file not found"},
		{name: "rejects missing required fields", filename: "cover.png", wantStatus: http.StatusUnprocessableEntity, message: "validation failed"},
		{name: "rejects unsupported image", filename: "cover.gif", fields: true, wantStatus: http.StatusUnprocessableEntity, message: "invalid file format"},
		{name: "reports upload failure", filename: "cover.png", fields: true, uploadErr: errors.New("cloudinary down"), wantStatus: http.StatusInternalServerError, message: "upload failed"},
		{name: "reports service failure", filename: "cover.png", fields: true, createErr: errors.New("database down"), wantStatus: http.StatusInternalServerError, message: "error occurred"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			service := &bookServiceStub{book: books.Book{ID: 12, Title: "Dune"}, createErr: tt.createErr}
			upload := &uploaderStub{
				uploadResult: &fileupload.FileUploadResult{SecureURL: "https://example.com/cover.png", PublicID: "cover"},
				uploadErr:    tt.uploadErr,
			}
			handler := NewBooks(service, upload)
			app, token := protectedTestApp(t, models.Admin, func(app *fiber.App, _ middlewares.JWTConfig) {
				app.Post("/books", middlewares.VerifyAdmin(), middlewares.ValidateBody(&books.BookRequest{}), handler.Create)
			})

			body, contentType := multipartBookRequest(t, tt.fields, tt.filename)
			request := httptest.NewRequest(http.MethodPost, "/books", body)
			request.Header.Set("Content-Type", contentType)
			request.Header.Set("Authorization", "Bearer "+token)
			response, err := app.Test(request)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer response.Body.Close()

			assertResponse(t, response, tt.wantStatus, tt.message)
			if tt.name == "creates book" && (service.createdBook == nil || service.createdBook.ImageID != "cover") {
				t.Errorf("uploaded image was not applied to the created book: %+v", service.createdBook)
			}
		})
	}
}

func multipartBookRequest(t *testing.T, fields bool, filename string) (*bytes.Buffer, string) {
	t.Helper()

	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	if fields {
		for key, value := range map[string]string{
			"title":       "Dune",
			"description": "A novel",
			"publisher":   "Ace",
			"year":        "1965",
			"stock":       "3",
			"category_id": "7",
		} {
			if err := writer.WriteField(key, value); err != nil {
				t.Fatalf("write form field: %v", err)
			}
		}
	}
	if filename != "" {
		file, err := writer.CreateFormFile("file", filename)
		if err != nil {
			t.Fatalf("create image file: %v", err)
		}
		if _, err := file.Write([]byte("image data")); err != nil {
			t.Fatalf("write image file: %v", err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}
	return body, writer.FormDataContentType()
}

func TestBooksDeleteFailureCases(t *testing.T) {
	for _, tt := range []struct {
		name       string
		path       string
		getErr     error
		deleteErr  error
		wantStatus int
		message    string
	}{
		{name: "rejects malformed ID", path: "/books/nope", wantStatus: http.StatusBadRequest, message: "invalid id"},
		{name: "reports lookup failure", path: "/books/9", getErr: errors.New("database down"), wantStatus: http.StatusInternalServerError, message: "error occurred"},
		{name: "reports service delete failure", path: "/books/9", deleteErr: errors.New("database down"), wantStatus: http.StatusInternalServerError, message: "error occurred"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			service := &bookServiceStub{book: books.Book{ID: 9}, getErr: tt.getErr, deleteErr: tt.deleteErr}
			handler := NewBooks(service, &uploaderStub{})
			app, token := protectedTestApp(t, models.Admin, func(app *fiber.App, _ middlewares.JWTConfig) {
				app.Delete("/books/:id", middlewares.VerifyAdmin(), handler.Delete)
			})
			request := httptestRequest(http.MethodDelete, tt.path, "")
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

func TestBooksUpdateFailureCases(t *testing.T) {
	for _, tt := range []struct {
		name       string
		path       string
		filename   string
		book       books.Book
		getErr     error
		uploadErr  error
		deleteErr  error
		updateErr  error
		wantStatus int
		message    string
		wantID     uint
	}{
		{name: "rejects malformed ID", path: "/books/nope", wantStatus: http.StatusBadRequest, message: "invalid id"},
		{name: "reports lookup failure", path: "/books/8", getErr: errors.New("database down"), wantStatus: http.StatusInternalServerError, message: "error occurred", wantID: 8},
		{name: "rejects unsupported image", path: "/books/8", filename: "cover.gif", book: books.Book{ID: 8, ImageID: "old"}, wantStatus: http.StatusUnprocessableEntity, message: "invalid file format", wantID: 8},
		{name: "reports upload failure", path: "/books/8", filename: "cover.png", book: books.Book{ID: 8}, uploadErr: errors.New("cloudinary down"), wantStatus: http.StatusInternalServerError, message: "upload failed", wantID: 8},
		{name: "reports old image deletion failure", path: "/books/8", filename: "cover.png", book: books.Book{ID: 8, ImageID: "old"}, deleteErr: errors.New("cloudinary down"), wantStatus: http.StatusInternalServerError, message: "old image failed to delete", wantID: 8},
		{name: "reports update service failure", path: "/books/8", book: books.Book{ID: 8}, updateErr: errors.New("database down"), wantStatus: http.StatusInternalServerError, message: "error occurred", wantID: 8},
	} {
		t.Run(tt.name, func(t *testing.T) {
			service := &bookServiceStub{book: tt.book, getErr: tt.getErr, updateErr: tt.updateErr}
			upload := &uploaderStub{
				uploadResult: &fileupload.FileUploadResult{SecureURL: "https://example.com/new.png", PublicID: "new-image"},
				uploadErr:    tt.uploadErr,
				deleteErr:    tt.deleteErr,
			}
			handler := NewBooks(service, upload)
			app, token := protectedTestApp(t, models.Admin, func(app *fiber.App, _ middlewares.JWTConfig) {
				app.Patch("/books/:id", middlewares.VerifyAdmin(), func(ctx fiber.Ctx) error {
					ctx.Locals("validatedBody", &books.BookRequest{Title: "Updated"})
					return ctx.Next()
				}, handler.Update)
			})

			request := httptestRequest(http.MethodPatch, tt.path, "")
			if tt.filename != "" {
				body, contentType := multipartRequestWithFile(t, tt.filename)
				request = httptest.NewRequest(http.MethodPatch, tt.path, body)
				request.Header.Set("Content-Type", contentType)
			}
			request.Header.Set("Authorization", "Bearer "+token)
			response, err := app.Test(request)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer response.Body.Close()

			assertResponse(t, response, tt.wantStatus, tt.message)
			if service.updatedID != 0 && service.updatedID != tt.wantID {
				t.Errorf("expected update ID %d, got %d", tt.wantID, service.updatedID)
			}
		})
	}
}
