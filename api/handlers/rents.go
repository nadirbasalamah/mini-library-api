package handlers

import (
	"fmt"
	"go-digital-library-fiber/api/middlewares"
	"go-digital-library-fiber/db/models"
	"go-digital-library-fiber/pkg/dtos"
	"go-digital-library-fiber/pkg/rajaongkir"
	"go-digital-library-fiber/pkg/utils"
	"go-digital-library-fiber/rents"
	"go-digital-library-fiber/users"
	"net/http"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"
)

type Rents struct {
	rents     rents.Service
	users     users.Service
	roService deliveryFeeService
}

type deliveryFeeService interface {
	GetDeliveryFee(req rajaongkir.GetFeeRequest) (float64, error)
}

func (r Rents) GetAll(ctx fiber.Ctx) error {
	page, _ := strconv.Atoi(ctx.Query("page"))
	limit, _ := strconv.Atoi(ctx.Query("limit"))
	sort := ctx.Query("sort")
	search := ctx.Query("search")

	pagination := utils.Pagination{
		Page:    page,
		Limit:   limit,
		Sort:    sort,
		Search:  search,
		Keyword: "courier",
	}

	rentsData, err := r.rents.GetAll(ctx.Context(), pagination)
	if err != nil {
		return ctx.Status(http.StatusInternalServerError).JSON(dtos.Response[any]{
			Status:  "failed",
			Message: "fetch rents failed",
		})
	}

	return ctx.Status(http.StatusOK).JSON(dtos.Response[[]rents.Rent]{
		Status:  "success",
		Message: "all rents",
		Data:    rentsData,
	})
}

func (r Rents) GetByUser(ctx fiber.Ctx) error {
	userID, err := middlewares.GetUserID(ctx)
	if err != nil {
		return ctx.Status(http.StatusUnauthorized).JSON(dtos.Response[any]{
			Status:  "failed",
			Message: "invalid token",
		})
	}

	rentsData, err := r.rents.GetByUser(ctx.Context(), uint(userID))
	if err != nil {
		return ctx.Status(http.StatusInternalServerError).JSON(dtos.Response[any]{
			Status:  "failed",
			Message: utils.GetErrorMessage(err),
		})
	}

	return ctx.Status(http.StatusOK).JSON(dtos.Response[[]rents.Rent]{
		Status:  "success",
		Message: "user rents",
		Data:    rentsData,
	})
}

func (r Rents) GetByID(ctx fiber.Ctx) error {
	id, err := strconv.ParseUint(ctx.Params("id"), 10, 64)
	if err != nil {
		return ctx.Status(http.StatusBadRequest).JSON(dtos.Response[any]{
			Status:  "failed",
			Message: "invalid id",
		})
	}

	rent, err := r.rents.GetByID(ctx.Context(), uint(id))
	if err != nil {
		return ctx.Status(http.StatusInternalServerError).JSON(dtos.Response[any]{
			Status:  "failed",
			Message: utils.GetErrorMessage(err),
		})
	}

	user, err := middlewares.GetUser(ctx)
	if err != nil {
		return ctx.Status(http.StatusUnauthorized).JSON(dtos.Response[any]{
			Status:  "failed",
			Message: "invalid token",
		})
	}
	if user.Role != models.Admin && rent.UserID != uint(user.ID) {
		return ctx.Status(http.StatusForbidden).JSON(dtos.Response[any]{
			Status:  "failed",
			Message: "access denied",
		})
	}

	return ctx.Status(http.StatusOK).JSON(dtos.Response[rents.Rent]{
		Status:  "success",
		Message: "rent found",
		Data:    rent,
	})
}

func (r Rents) Create(ctx fiber.Ctx) error {
	rentReq := ctx.Locals("validatedBody").(*rents.RentRequest)

	userID, err := middlewares.GetUserID(ctx)
	if err != nil {
		return ctx.Status(http.StatusUnauthorized).JSON(dtos.Response[any]{
			Status:  "failed",
			Message: "invalid token",
		})
	}

	user, err := r.users.GetProfile(ctx.Context(), uint(userID))

	fmt.Println(user.DistrictID)

	if err != nil {
		return ctx.Status(http.StatusInternalServerError).JSON(dtos.Response[any]{
			Status:  "failed",
			Message: utils.GetErrorMessage(err),
		})
	}

	rentReq.UserID = uint(userID)
	rentReq.ReturnTime = time.Now().AddDate(0, 0, int(rentReq.Duration))

	fee, err := r.roService.GetDeliveryFee(rajaongkir.GetFeeRequest{
		Origin:      "1347",
		Destination: strconv.Itoa(int(user.DistrictID)),
		Courier:     rentReq.Courier,
	})

	if err != nil {
		return ctx.Status(http.StatusInternalServerError).JSON(dtos.Response[any]{
			Status:  "failed",
			Message: err.Error(),
		})
	}

	rentReq.Fee = fee

	if err := r.rents.Create(ctx.Context(), rentReq); err != nil {
		return ctx.Status(http.StatusInternalServerError).JSON(dtos.Response[any]{
			Status:  "failed",
			Message: utils.GetErrorMessage(err),
		})
	}

	return ctx.Status(http.StatusCreated).JSON(dtos.Response[any]{
		Status:  "success",
		Message: "rent created",
	})
}

func (r Rents) Update(ctx fiber.Ctx) error {
	id, err := strconv.ParseUint(ctx.Params("id"), 10, 64)
	if err != nil {
		return ctx.Status(http.StatusBadRequest).JSON(dtos.Response[any]{
			Status:  "failed",
			Message: "invalid id",
		})
	}

	rentReq := ctx.Locals("validatedBody").(*rents.UpdateRentRequest)
	rent, err := r.rents.Update(ctx.Context(), rentReq, uint(id))
	if err != nil {
		return ctx.Status(http.StatusInternalServerError).JSON(dtos.Response[any]{
			Status:  "failed",
			Message: utils.GetErrorMessage(err),
		})
	}

	return ctx.Status(http.StatusOK).JSON(dtos.Response[rents.Rent]{
		Status:  "success",
		Message: "rent updated",
		Data:    rent,
	})
}

func (r Rents) Delete(ctx fiber.Ctx) error {
	id, err := strconv.ParseUint(ctx.Params("id"), 10, 64)
	if err != nil {
		return ctx.Status(http.StatusBadRequest).JSON(dtos.Response[any]{
			Status:  "failed",
			Message: "invalid id",
		})
	}

	if err := r.rents.Delete(ctx.Context(), uint(id)); err != nil {
		return ctx.Status(http.StatusInternalServerError).JSON(dtos.Response[any]{
			Status:  "failed",
			Message: utils.GetErrorMessage(err),
		})
	}

	return ctx.Status(http.StatusOK).JSON(dtos.Response[any]{
		Status:  "success",
		Message: "rent deleted",
	})
}

func NewRents(rentService rents.Service, userService users.Service, roService deliveryFeeService) Rents {
	return Rents{
		rents:     rentService,
		roService: roService,
		users:     userService,
	}
}
