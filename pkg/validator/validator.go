package validator

import "github.com/go-playground/validator/v10"

var v = validator.New()

func Validate(val any) error {
	return v.Struct(val)
}
