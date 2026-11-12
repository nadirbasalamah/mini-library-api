package handlers

import (
	"bytes"
	"context"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-digital-library-fiber/api/middlewares"
	"go-digital-library-fiber/db/models"
	"go-digital-library-fiber/pkg/fileupload"
	"go-digital-library-fiber/users"

	"github.com/gofiber/fiber/v3"
)

type userServiceStub struct {
	profile   users.User
	updated   users.User
	getErr    error
	updateErr error
	request   *users.EditProfileRequest
	updatedID uint
}

func (s *userServiceStub) GetProfile(context.Context, uint) (users.User, error) {
	return s.profile, s.getErr
}

func (s *userServiceStub) Update(_ context.Context, request *users.EditProfileRequest, id uint) (users.User, error) {
	s.request = request
	s.updatedID = id
	return s.updated, s.updateErr
}

func TestUsersGetProfileCases(t *testing.T) {
	for _, tt := range []struct {
		name       string
		err        error
		wantStatus int
		message    string
	}{
		{name: "returns profile", wantStatus: http.StatusOK, message: "user data"},
		{name: "reports missing profile", err: errors.New("profile lookup failed"), wantStatus: http.StatusNotFound, message: "error occurred"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			service := &userServiceStub{profile: users.User{ID: 42, Username: "reader1"}, getErr: tt.err}
			handler := NewUsers(service, &uploaderStub{})
			app, token := protectedTestApp(t, models.Enduser, func(app *fiber.App, _ middlewares.JWTConfig) {
				app.Get("/profile", handler.GetProfile)
			})
			request := httptestRequest(http.MethodGet, "/profile", "")
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

func TestUsersEditProfileCases(t *testing.T) {
	t.Run("updates fields and preserves picture", func(t *testing.T) {
		service := &userServiceStub{
			profile: users.User{ID: 42, ProfilePicture: "https://example.com/current.png", ProfilePictureID: "current"},
			updated: users.User{ID: 42, Username: "reader2", ProfilePicture: "https://example.com/current.png"},
		}
		handler := NewUsers(service, &uploaderStub{})
		app, token := protectedTestApp(t, models.Enduser, func(app *fiber.App, _ middlewares.JWTConfig) {
			app.Patch("/profile/edit", handler.EditProfile)
		})
		request := jsonRequest(http.MethodPatch, "/profile/edit", `{"username":"reader2","email":"reader2@example.com","address":"2 Main St"}`)
		request.Header.Set("Authorization", "Bearer "+token)
		response, err := app.Test(request)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer response.Body.Close()

		assertResponse(t, response, http.StatusOK, "profile updated")
		if service.updatedID != 42 || service.request == nil || service.request.ProfilePictureID != "current" {
			t.Errorf("update did not preserve the current picture: ID=%d request=%+v", service.updatedID, service.request)
		}
	})

	t.Run("replaces profile picture", func(t *testing.T) {
		service := &userServiceStub{
			profile: users.User{ID: 42, ProfilePictureID: "old-picture"},
			updated: users.User{ID: 42, ProfilePicture: "https://example.com/new.png"},
		}
		upload := &uploaderStub{uploadResult: &fileupload.FileUploadResult{SecureURL: "https://example.com/new.png", PublicID: "new-picture"}}
		handler := NewUsers(service, upload)
		app, token := protectedTestApp(t, models.Enduser, func(app *fiber.App, _ middlewares.JWTConfig) {
			app.Patch("/profile/edit", handler.EditProfile)
		})
		body, contentType := multipartProfileRequest(t, "reader2", "reader2@example.com", "2 Main St", "avatar.png")
		request := httptest.NewRequest(http.MethodPatch, "/profile/edit", bytes.NewReader(body.Bytes()))
		request.Header.Set("Content-Type", contentType)
		request.Header.Set("Authorization", "Bearer "+token)
		response, err := app.Test(request)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer response.Body.Close()

		assertResponse(t, response, http.StatusOK, "profile updated")
		if service.request == nil || service.request.ProfilePicture != "https://example.com/new.png" || service.request.ProfilePictureID != "new-picture" {
			t.Errorf("replacement picture was not applied: %+v", service.request)
		}
		if len(upload.deletedIDs) != 1 || upload.deletedIDs[0] != "old-picture" {
			t.Errorf("expected old image deletion, got %v", upload.deletedIDs)
		}
	})

	for _, tt := range []struct {
		name       string
		body       string
		filename   string
		getErr     error
		updateErr  error
		uploadErr  error
		deleteErr  error
		wantStatus int
		message    string
	}{
		{name: "rejects invalid fields", body: `{"username":"","email":"bad-email","address":""}`, wantStatus: http.StatusUnprocessableEntity, message: "validation failed"},
		{name: "reports user lookup failure", body: `{"username":"reader2","email":"reader2@example.com","address":"2 Main St"}`, getErr: errors.New("missing"), wantStatus: http.StatusNotFound, message: "user not found"},
		{name: "rejects unsupported image", filename: "avatar.gif", wantStatus: http.StatusUnprocessableEntity, message: "invalid file format"},
		{name: "reports upload failure", filename: "avatar.png", uploadErr: errors.New("cloudinary down"), wantStatus: http.StatusInternalServerError, message: "upload failed"},
		{name: "reports old image deletion failure", filename: "avatar.png", deleteErr: errors.New("cloudinary down"), wantStatus: http.StatusInternalServerError, message: "old file failed to delete"},
		{name: "reports update failure", body: `{"username":"reader2","email":"reader2@example.com","address":"2 Main St"}`, updateErr: errors.New("database down"), wantStatus: http.StatusInternalServerError, message: "error occurred"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			service := &userServiceStub{
				profile:   users.User{ID: 42, ProfilePictureID: "old-picture"},
				updated:   users.User{ID: 42},
				getErr:    tt.getErr,
				updateErr: tt.updateErr,
			}
			upload := &uploaderStub{
				uploadResult: &fileupload.FileUploadResult{SecureURL: "https://example.com/new.png", PublicID: "new-picture"},
				uploadErr:    tt.uploadErr,
				deleteErr:    tt.deleteErr,
			}
			handler := NewUsers(service, upload)
			app, token := protectedTestApp(t, models.Enduser, func(app *fiber.App, _ middlewares.JWTConfig) {
				app.Patch("/profile/edit", handler.EditProfile)
			})

			var request *http.Request
			if tt.filename != "" {
				body, contentType := multipartProfileRequest(t, "reader2", "reader2@example.com", "2 Main St", tt.filename)
				request = httptest.NewRequest(http.MethodPatch, "/profile/edit", bytes.NewReader(body.Bytes()))
				request.Header.Set("Content-Type", contentType)
			} else {
				request = jsonRequest(http.MethodPatch, "/profile/edit", tt.body)
			}
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

func multipartProfileRequest(t *testing.T, username, email, address, filename string) (*bytes.Buffer, string) {
	t.Helper()

	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	for key, value := range map[string]string{
		"username": username,
		"email":    email,
		"address":  address,
	} {
		if err := writer.WriteField(key, value); err != nil {
			t.Fatalf("write form field: %v", err)
		}
	}
	file, err := writer.CreateFormFile("file", filename)
	if err != nil {
		t.Fatalf("create upload: %v", err)
	}
	if _, err := file.Write([]byte("image data")); err != nil {
		t.Fatalf("write upload: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}
	return body, writer.FormDataContentType()
}
