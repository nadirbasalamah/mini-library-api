package handlers

import (
	"go-digital-library-fiber/categories"
	"go-digital-library-fiber/pkg/dtos"
	"go-digital-library-fiber/pkg/utils"
	"net/http"
	"strconv"

	"github.com/gofiber/fiber/v3"
)

type Categories struct {
	categories categories.Service
}

func (c Categories) GetAll(ctx fiber.Ctx) error {
	page, _ := strconv.Atoi(ctx.Query("page"))
	limit, _ := strconv.Atoi(ctx.Query("limit"))
	sort := ctx.Query("sort")
	search := ctx.Query("search")

	pagination := utils.Pagination{
		Page:    page,
		Limit:   limit,
		Sort:    sort,
		Search:  search,
		Keyword: "name",
	}

	categoriesData, err := c.categories.GetAll(ctx.Context(), pagination)

	if err != nil {
		return ctx.Status(http.StatusInternalServerError).JSON(dtos.Response[any]{
			Status:  "failed",
			Message: "fetch category failed",
		})
	}

	return ctx.Status(http.StatusOK).JSON(dtos.Response[utils.Pagination]{
		Status:  "success",
		Message: "all categories",
		Data:    categoriesData,
	})
}

func (c Categories) GetByID(ctx fiber.Ctx) error {
	param := ctx.Params("id")

	id, err := strconv.ParseUint(param, 10, 64)

	if err != nil {
		return ctx.Status(http.StatusBadRequest).JSON(dtos.Response[any]{
			Status:  "failed",
			Message: "invalid id",
		})
	}

	category, err := c.categories.GetByID(ctx.Context(), uint(id))

	if err != nil {
		return ctx.Status(http.StatusInternalServerError).JSON(dtos.Response[any]{
			Status:  "failed",
			Message: utils.GetErrorMessage(err),
		})
	}

	return ctx.Status(http.StatusOK).JSON(dtos.Response[categories.Category]{
		Status:  "success",
		Message: "category found",
		Data:    category,
	})
}

func (c Categories) Create(ctx fiber.Ctx) error {
	categoryReq := ctx.Locals("validatedBody").(*categories.CategoryRequest)

	category, err := c.categories.Create(ctx.Context(), categoryReq)

	if err != nil {
		return ctx.Status(http.StatusInternalServerError).JSON(dtos.Response[any]{
			Status:  "failed",
			Message: utils.GetErrorMessage(err),
		})
	}

	return ctx.Status(http.StatusCreated).JSON(dtos.Response[any]{
		Status:  "success",
		Message: "category created",
		Data:    category,
	})
}

func (c Categories) Update(ctx fiber.Ctx) error {
	param := ctx.Params("id")

	id, err := strconv.ParseUint(param, 10, 64)

	if err != nil {
		return ctx.Status(http.StatusBadRequest).JSON(dtos.Response[any]{
			Status:  "failed",
			Message: "invalid id",
		})
	}

	categoryReq := ctx.Locals("validatedBody").(*categories.CategoryRequest)

	category, err := c.categories.Update(ctx.Context(), categoryReq, uint(id))

	if err != nil {
		return ctx.Status(http.StatusInternalServerError).JSON(dtos.Response[any]{
			Status:  "failed",
			Message: utils.GetErrorMessage(err),
		})
	}

	return ctx.Status(http.StatusOK).JSON(dtos.Response[categories.Category]{
		Status:  "success",
		Message: "category updated",
		Data:    category,
	})
}

func (c Categories) Delete(ctx fiber.Ctx) error {
	param := ctx.Params("id")

	id, err := strconv.ParseUint(param, 10, 64)

	if err != nil {
		return ctx.Status(http.StatusBadRequest).JSON(dtos.Response[any]{
			Status:  "failed",
			Message: "invalid id",
		})
	}

	err = c.categories.Delete(ctx.Context(), uint(id))

	if err != nil {
		return ctx.Status(http.StatusInternalServerError).JSON(dtos.Response[any]{
			Status:  "failed",
			Message: utils.GetErrorMessage(err),
		})
	}

	return ctx.Status(http.StatusOK).JSON(dtos.Response[any]{
		Status:  "success",
		Message: "category deleted",
	})
}

func NewCategories(categories categories.Service) Categories {
	categoriesHandler := Categories{
		categories: categories,
	}

	return categoriesHandler
}
