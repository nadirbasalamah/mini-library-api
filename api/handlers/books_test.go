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

	"go-digital-library-fiber/books"
	"go-digital-library-fiber/pkg/fileupload"
	"go-digital-library-fiber/pkg/utils"

	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"github.com/gofiber/fiber/v3"
)

type bookServiceStub struct {
	categoryPagination utils.Pagination
	categoryID         uint
	book               books.Book
	updatedBook        *books.BookRequest
	updatedID          uint
	deletedID          uint
}

func (s *bookServiceStub) GetAll(context.Context, utils.Pagination) (utils.Pagination, error) {
	return utils.Pagination{}, nil
}

func (s *bookServiceStub) GetByID(context.Context, uint) (books.Book, error) {
	return s.book, nil
}

func (s *bookServiceStub) GetByCategory(_ context.Context, pagination utils.Pagination, categoryID uint) (utils.Pagination, error) {
	s.categoryPagination = pagination
	s.categoryID = categoryID
	return utils.Pagination{Rows: []books.Book{{ID: 1, Title: "Dune"}}}, nil
}

func (s *bookServiceStub) Create(context.Context, *books.BookRequest) (books.Book, error) {
	return books.Book{}, nil
}

func (s *bookServiceStub) Update(_ context.Context, request *books.BookRequest, id uint) (books.Book, error) {
	s.updatedBook = request
	s.updatedID = id
	return s.book, nil
}

func (s *bookServiceStub) Delete(_ context.Context, id uint) error {
	s.deletedID = id
	return nil
}

type uploaderStub struct {
	uploadResult *fileupload.FileUploadResult
	uploaded     bool
	deletedIDs   []string
	deleteErr    error
}

func (s *uploaderStub) UploadFile(context.Context, *multipart.FileHeader, uploader.UploadParams) (*fileupload.FileUploadResult, error) {
	s.uploaded = true
	return s.uploadResult, nil
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
