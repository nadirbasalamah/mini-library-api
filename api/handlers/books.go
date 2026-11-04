package handlers

import (
	"go-digital-library-fiber/books"
	"go-digital-library-fiber/pkg/constant"
	"go-digital-library-fiber/pkg/dtos"
	"go-digital-library-fiber/pkg/fileupload"
	"go-digital-library-fiber/pkg/utils"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"github.com/gofiber/fiber/v3"
)

type Books struct {
	books    books.Service
	uploader fileupload.Uploader
}

func (b Books) GetAll(ctx fiber.Ctx) error {
	page, _ := strconv.Atoi(ctx.Query("page"))
	limit, _ := strconv.Atoi(ctx.Query("limit"))
	sort := ctx.Query("sort")
	search := ctx.Query("search")

	pagination := utils.Pagination{
		Page:    page,
		Limit:   limit,
		Sort:    sort,
		Search:  search,
		Keyword: "title",
	}

	booksData, err := b.books.GetAll(ctx.Context(), pagination)
	if err != nil {
		return ctx.Status(http.StatusInternalServerError).JSON(dtos.Response[any]{
			Status:  "failed",
			Message: "fetch books failed",
		})
	}

	return ctx.Status(http.StatusOK).JSON(dtos.Response[utils.Pagination]{
		Status:  "success",
		Message: "all books",
		Data:    booksData,
	})
}

func (b Books) GetByID(ctx fiber.Ctx) error {
	id, err := strconv.ParseUint(ctx.Params("id"), 10, 64)
	if err != nil {
		return ctx.Status(http.StatusBadRequest).JSON(dtos.Response[any]{
			Status:  "failed",
			Message: "invalid id",
		})
	}

	book, err := b.books.GetByID(ctx.Context(), uint(id))
	if err != nil {
		return ctx.Status(http.StatusInternalServerError).JSON(dtos.Response[any]{
			Status:  "failed",
			Message: utils.GetErrorMessage(err),
		})
	}

	return ctx.Status(http.StatusOK).JSON(dtos.Response[books.Book]{
		Status:  "success",
		Message: "book found",
		Data:    book,
	})
}

func (b Books) GetByCategory(ctx fiber.Ctx) error {
	id, err := strconv.ParseUint(ctx.Params("id"), 10, 64)
	if err != nil {
		return ctx.Status(http.StatusBadRequest).JSON(dtos.Response[any]{
			Status:  "failed",
			Message: "invalid id",
		})
	}

	page, _ := strconv.Atoi(ctx.Query("page"))
	limit, _ := strconv.Atoi(ctx.Query("limit"))
	sort := ctx.Query("sort")
	search := ctx.Query("search")

	pagination := utils.Pagination{
		Page:    page,
		Limit:   limit,
		Sort:    sort,
		Search:  search,
		Keyword: "title",
	}

	booksData, err := b.books.GetByCategory(ctx.Context(), pagination, uint(id))
	if err != nil {
		return ctx.Status(http.StatusInternalServerError).JSON(dtos.Response[any]{
			Status:  "failed",
			Message: utils.GetErrorMessage(err),
		})
	}

	return ctx.Status(http.StatusOK).JSON(dtos.Response[utils.Pagination]{
		Status:  "success",
		Message: "books by category",
		Data:    booksData,
	})
}

func (b Books) Create(ctx fiber.Ctx) error {
	bookReq := ctx.Locals("validatedBody").(*books.BookRequest)

	file, err := ctx.FormFile("file")

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

	resp, err := b.uploader.UploadFile(ctx.Context(), file, uploader.UploadParams{})

	if err != nil {
		return ctx.Status(http.StatusInternalServerError).JSON(dtos.Response[any]{
			Status:  "failed",
			Message: "upload failed",
		})
	}

	bookReq.ImageLink = resp.SecureURL
	bookReq.ImageID = resp.PublicID

	book, err := b.books.Create(ctx.Context(), bookReq)
	if err != nil {
		return ctx.Status(http.StatusInternalServerError).JSON(dtos.Response[any]{
			Status:  "failed",
			Message: utils.GetErrorMessage(err),
		})
	}

	return ctx.Status(http.StatusCreated).JSON(dtos.Response[books.Book]{
		Status:  "success",
		Message: "book created",
		Data:    book,
	})
}

func (b Books) Update(ctx fiber.Ctx) error {
	id, err := strconv.ParseUint(ctx.Params("id"), 10, 64)
	if err != nil {
		return ctx.Status(http.StatusBadRequest).JSON(dtos.Response[any]{
			Status:  "failed",
			Message: "invalid id",
		})
	}

	bookReq := ctx.Locals("validatedBody").(*books.BookRequest)

	existingBook, err := b.books.GetByID(ctx.Context(), uint(id))
	if err != nil {
		return ctx.Status(http.StatusInternalServerError).JSON(dtos.Response[any]{
			Status:  "failed",
			Message: utils.GetErrorMessage(err),
		})
	}

	file, _ := ctx.FormFile("file")
	bookReq.ImageLink = existingBook.ImageLink
	bookReq.ImageID = existingBook.ImageID

	if file != nil {
		ext := strings.ToLower(filepath.Ext(file.Filename))
		if !utils.ValidateFile(ext) {
			return ctx.Status(http.StatusUnprocessableEntity).JSON(dtos.Response[any]{
				Status:  "failed",
				Message: "invalid file format",
			})
		}

		resp, err := b.uploader.UploadFile(ctx.Context(), file, uploader.UploadParams{})
		if err != nil || resp == nil {
			return ctx.Status(http.StatusInternalServerError).JSON(dtos.Response[any]{
				Status:  "failed",
				Message: "upload failed",
			})
		}

		if existingBook.ImageID != "" {
			if _, err := b.uploader.DeleteFile(ctx.Context(), existingBook.ImageID, constant.RESOURCE_IMAGE, uploader.UploadParams{}); err != nil {
				return ctx.Status(http.StatusInternalServerError).JSON(dtos.Response[any]{
					Status:  "failed",
					Message: "old image failed to delete",
				})
			}
		}

		bookReq.ImageLink = resp.SecureURL
		bookReq.ImageID = resp.PublicID
	}

	book, err := b.books.Update(ctx.Context(), bookReq, uint(id))
	if err != nil {
		return ctx.Status(http.StatusInternalServerError).JSON(dtos.Response[any]{
			Status:  "failed",
			Message: utils.GetErrorMessage(err),
		})
	}

	return ctx.Status(http.StatusOK).JSON(dtos.Response[books.Book]{
		Status:  "success",
		Message: "book updated",
		Data:    book,
	})
}

func (b Books) Delete(ctx fiber.Ctx) error {
	id, err := strconv.ParseUint(ctx.Params("id"), 10, 64)
	if err != nil {
		return ctx.Status(http.StatusBadRequest).JSON(dtos.Response[any]{
			Status:  "failed",
			Message: "invalid id",
		})
	}

	book, err := b.books.GetByID(ctx.Context(), uint(id))
	if err != nil {
		return ctx.Status(http.StatusInternalServerError).JSON(dtos.Response[any]{
			Status:  "failed",
			Message: utils.GetErrorMessage(err),
		})
	}

	if book.ImageID != "" {
		if _, err := b.uploader.DeleteFile(ctx.Context(), book.ImageID, constant.RESOURCE_IMAGE, uploader.UploadParams{}); err != nil {
			return ctx.Status(http.StatusInternalServerError).JSON(dtos.Response[any]{
				Status:  "failed",
				Message: "image failed to delete",
			})
		}
	}

	if err := b.books.Delete(ctx.Context(), uint(id)); err != nil {
		return ctx.Status(http.StatusInternalServerError).JSON(dtos.Response[any]{
			Status:  "failed",
			Message: utils.GetErrorMessage(err),
		})
	}

	return ctx.Status(http.StatusOK).JSON(dtos.Response[any]{
		Status:  "success",
		Message: "book deleted",
	})
}

func NewBooks(bookService books.Service, uploader fileupload.Uploader) Books {
	return Books{
		books:    bookService,
		uploader: uploader,
	}
}
