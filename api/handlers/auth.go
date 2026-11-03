package handlers

import (
	"go-digital-library-fiber/api/middlewares"
	"go-digital-library-fiber/auth"
	"go-digital-library-fiber/db/models"
	"go-digital-library-fiber/pkg/dtos"
	"go-digital-library-fiber/pkg/utils"
	"net/http"

	"github.com/gofiber/fiber/v3"
)

type Auth struct {
	auth      auth.Service
	jwtConfig middlewares.JWTConfig
}

func (a Auth) Register(ctx fiber.Ctx) error {
	registerReq := new(auth.RegisterRequest)

	if err := ctx.Bind().JSON(registerReq); err != nil {
		return ctx.Status(http.StatusBadRequest).JSON(dtos.Response[any]{
			Status:  "failed",
			Message: "invalid request",
		})
	}

	if err := utils.ValidateStruct(registerReq); err != nil {
		return ctx.Status(http.StatusUnprocessableEntity).JSON(dtos.Response[any]{
			Status:  "failed",
			Message: "validation failed",
			Data:    utils.GetValidationErrMessages(err.Error()),
		})
	}

	user, err := a.auth.Register(ctx.Context(), registerReq)

	if err != nil {
		return ctx.Status(http.StatusInternalServerError).JSON(dtos.Response[any]{
			Status:  "failed",
			Message: utils.GetErrorMessage(err),
		})
	}

	return ctx.Status(http.StatusCreated).JSON(dtos.Response[auth.User]{
		Status:  "success",
		Message: "user registered",
		Data:    user,
	})
}

func (a Auth) Login(ctx fiber.Ctx) error {
	loginReq := new(auth.LoginRequest)

	if err := ctx.Bind().JSON(loginReq); err != nil {
		return ctx.Status(http.StatusBadRequest).JSON(dtos.Response[any]{
			Status:  "failed",
			Message: "invalid request",
		})
	}

	if err := utils.ValidateStruct(loginReq); err != nil {
		return ctx.Status(http.StatusUnprocessableEntity).JSON(dtos.Response[any]{
			Status:  "failed",
			Message: "validation failed",
			Data:    utils.GetValidationErrMessages(err.Error()),
		})
	}

	user, err := a.auth.Login(ctx.Context(), loginReq)

	if err != nil {
		return ctx.Status(http.StatusInternalServerError).JSON(dtos.Response[any]{
			Status:  "failed",
			Message: utils.GetErrorMessage(err),
		})
	}

	token, err := a.jwtConfig.GenerateToken(int(user.ID), models.Role(user.Role))

	if err != nil {
		return ctx.Status(http.StatusInternalServerError).JSON(dtos.Response[any]{
			Status:  "failed",
			Message: "token generation failed",
		})
	}

	return ctx.Status(http.StatusOK).JSON(dtos.Response[string]{
		Status:  "success",
		Message: "login success",
		Data:    token,
	})
}

func NewAuth(auth auth.Service, jwtConfig middlewares.JWTConfig) Auth {
	authHandler := Auth{
		auth:      auth,
		jwtConfig: jwtConfig,
	}

	return authHandler
}
