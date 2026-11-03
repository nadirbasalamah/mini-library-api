package utils

import (
	"go-digital-library-fiber/pkg/constant"
	"regexp"

	"github.com/go-playground/validator/v10"
)

var validate *validator.Validate

func init() {
	validate = validator.New()

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
}

func ValidateFile(ext string) bool {
	return constant.ALLOWED_EXTENSIONS[ext]
}

func ValidateStruct(s interface{}) error {
	return validate.Struct(s)
}
