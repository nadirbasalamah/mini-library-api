package middlewares

import (
	"errors"
	"go-digital-library-fiber/pkg/dtos"
	"go-digital-library-fiber/pkg/rajaongkir"
	"go-digital-library-fiber/pkg/utils"
	"net/http"
	"reflect"
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
)

type CustomValidator struct {
	Validator *validator.Validate
}

func InitValidator() *validator.Validate {
	validate := validator.New()

	validate.RegisterValidation("containsNumber", func(fl validator.FieldLevel) bool {
		match, _ := regexp.MatchString(`[0-9]`, fl.Field().String())
		return match
	})

	validate.RegisterValidation("containsUppercaseLetter", func(fl validator.FieldLevel) bool {
		match, _ := regexp.MatchString(`[A-Z]`, fl.Field().String())
		return match
	})

	validate.RegisterValidation("containsLowercaseLetter", func(fl validator.FieldLevel) bool {
		match, _ := regexp.MatchString(`[a-z]`, fl.Field().String())
		return match
	})

	validate.RegisterValidation("containsSpecialCharacter", func(fl validator.FieldLevel) bool {
		match, _ := regexp.MatchString(`[!@#$%^&*()]`, fl.Field().String())
		return match
	})

	validate.RegisterValidation("validCourier", func(fl validator.FieldLevel) bool {
		_, ok := rajaongkir.COURIERS[fl.Field().String()]
		return ok
	})

	return validate
}

func (cv *CustomValidator) Validate(i interface{}) error {
	var sb strings.Builder

	if err := cv.Validator.Struct(i); err != nil {
		for _, err := range err.(validator.ValidationErrors) {
			sb.WriteString(getErrorMessage(err.Field(), err))
			sb.WriteString(",")
		}

		return errors.New(sb.String())
	}
	return nil
}

func ValidateBody(dto any) fiber.Handler {
	return func(c fiber.Ctx) error {
		customValidator := CustomValidator{
			Validator: InitValidator(),
		}
		dtoType := reflect.TypeOf(dto).Elem()
		req := reflect.New(dtoType).Interface()

		if err := c.Bind().All(req); err != nil {
			return c.Status(http.StatusBadRequest).JSON(dtos.Response[any]{
				Status:  "failed",
				Message: "invalid request body",
				Data:    err.Error(),
			})
		}

		if err := customValidator.Validate(req); err != nil {
			return c.Status(http.StatusUnprocessableEntity).JSON(dtos.Response[any]{
				Status:  "failed",
				Message: "validation failed",
				Data:    utils.GetValidationErrMessages(err.Error()),
			})
		}

		c.Locals("validatedBody", req)

		return c.Next()
	}
}

func getErrorMessage(fieldName string, err validator.FieldError) string {
	switch err.Tag() {
	case "required":
		return fieldName + " is required"
	case "email":
		return "the email is invalid"
	case "min":
		return "the minimum length of " + fieldName + " is equals " + err.Param()
	case "max":
		return "the maximum length of " + fieldName + " is equals " + err.Param()
	case "containsNumber":
		return "the " + fieldName + " must contains number"
	case "containsUppercaseLetter":
		return "the " + fieldName + " must contains uppercase letter"
	case "containsLowercaseLetter":
		return "the " + fieldName + " must contains lowercase letter"
	case "containsSpecialCharacter":
		return "the " + fieldName + " must contains special character"
	case "validCourier":
		return "invalid courier"
	case "gte":
		return "the " + fieldName + " must be greater than or equal to 1"
	default:
		return "validation error in " + fieldName
	}
}
