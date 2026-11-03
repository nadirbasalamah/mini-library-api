package handlers

import (
	"go-digital-library-fiber/api/middlewares"
	"go-digital-library-fiber/pkg/constant"
	"go-digital-library-fiber/pkg/dtos"
	"go-digital-library-fiber/pkg/fileupload"
	"go-digital-library-fiber/pkg/utils"
	usersPkg "go-digital-library-fiber/users"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"github.com/gofiber/fiber/v3"
)

type Users struct {
	users    usersPkg.Service
	uploader fileupload.Uploader
}

func (u Users) GetProfile(ctx fiber.Ctx) error {
	userID, err := middlewares.GetUserID(ctx)

	if err != nil {
		return ctx.Status(http.StatusUnauthorized).JSON(dtos.Response[any]{
			Status:  "failed",
			Message: "invalid token",
		})
	}

	user, err := u.users.GetProfile(ctx.Context(), uint(userID))

	if err != nil {
		return ctx.Status(http.StatusNotFound).JSON(dtos.Response[any]{
			Status:  "failed",
			Message: utils.GetErrorMessage(err),
		})
	}

	return ctx.Status(http.StatusOK).JSON(dtos.Response[usersPkg.User]{
		Status:  "success",
		Message: "user data",
		Data:    user,
	})
}

func (u Users) EditProfile(ctx fiber.Ctx) error {
	userID, err := middlewares.GetUserID(ctx)

	if err != nil {
		return ctx.Status(http.StatusUnauthorized).JSON(dtos.Response[any]{
			Status:  "failed",
			Message: "invalid token",
		})
	}

	editReq := new(usersPkg.EditProfileRequest)

	if err := ctx.Bind().All(editReq); err != nil {
		return ctx.Status(http.StatusBadRequest).JSON(dtos.Response[any]{
			Status:  "failed",
			Message: "invalid request",
		})
	}

	if err := utils.ValidateStruct(editReq); err != nil {
		return ctx.Status(http.StatusUnprocessableEntity).JSON(dtos.Response[any]{
			Status:  "failed",
			Message: "validation failed",
			Data:    utils.GetValidationErrMessages(err.Error()),
		})
	}

	userRecord, err := u.users.GetProfile(ctx.Context(), uint(userID))

	if err != nil {
		return ctx.Status(http.StatusNotFound).JSON(dtos.Response[any]{
			Status:  "failed",
			Message: "user not found",
		})
	}

	file, err := ctx.FormFile("file")

	if file != nil {
		if err != nil {
			return ctx.Status(http.StatusUnprocessableEntity).JSON(dtos.Response[any]{
				Status:  "failed",
				Message: "file not found",
			})
		}

		ext := strings.ToLower(filepath.Ext(file.Filename))

		isFileValid := utils.ValidateFile(ext)

		if !isFileValid {
			return ctx.Status(http.StatusUnprocessableEntity).JSON(dtos.Response[any]{
				Status:  "failed",
				Message: "invalid file format",
			})
		}

		resp, err := u.uploader.UploadFile(ctx.Context(), file, uploader.UploadParams{})

		if err != nil {
			return ctx.Status(http.StatusInternalServerError).JSON(dtos.Response[any]{
				Status:  "failed",
				Message: "upload failed",
			})
		}

		editReq.ProfilePicture = resp.SecureURL
		editReq.ProfilePictureID = resp.PublicID

		if _, err := u.uploader.DeleteFile(ctx.Context(), userRecord.ProfilePictureID, constant.RESOURCE_IMAGE, uploader.UploadParams{}); err != nil {
			return ctx.Status(http.StatusInternalServerError).JSON(dtos.Response[any]{
				Status:  "update failed",
				Message: "old file failed to delete",
			})
		}

	} else {
		editReq.ProfilePicture = userRecord.ProfilePicture
		editReq.ProfilePictureID = userRecord.ProfilePictureID
	}

	user, err := u.users.Update(ctx.Context(), editReq, uint(userID))

	if err != nil {
		return ctx.Status(http.StatusInternalServerError).JSON(dtos.Response[any]{
			Status:  "failed",
			Message: utils.GetErrorMessage(err),
		})
	}

	return ctx.Status(http.StatusOK).JSON(dtos.Response[usersPkg.User]{
		Status:  "success",
		Message: "profile updated",
		Data:    user,
	})
}

func NewUsers(users usersPkg.Service, uploader fileupload.Uploader) Users {
	usersHandler := Users{
		users:    users,
		uploader: uploader,
	}

	return usersHandler
}
