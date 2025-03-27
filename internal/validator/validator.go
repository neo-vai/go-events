package validator

import (
	"encoding/json"
	"regexp"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

var (
	alphanumdashRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
)

// RegisterCustomValidators registers custom validation functions with Gin's validator.
func RegisterCustomValidators() {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterValidation("json", validateJSON)
		v.RegisterValidation("strongpassword", validateStrongPassword)
		v.RegisterValidation("alphanumdash", validateAlphanumDash)
	}
}

func validateJSON(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	if value == "" {
		return true // allow empty
	}
	var js json.RawMessage
	return json.Unmarshal([]byte(value), &js) == nil
}

func validateStrongPassword(fl validator.FieldLevel) bool {
	password := fl.Field().String()
	if len(password) < 8 {
		return false
	}
	hasLetter := regexp.MustCompile(`[a-zA-Z]`).MatchString(password)
	hasDigit := regexp.MustCompile(`[0-9]`).MatchString(password)
	return hasLetter && hasDigit
}

func validateAlphanumDash(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	return alphanumdashRegex.MatchString(value)
}
