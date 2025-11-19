package config

import (
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

var validate *validator.Validate

func Validator() *validator.Validate {
	if validate == nil {
		validate = validator.New(validator.WithRequiredStructEnabled())
		validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
			if s := fld.Tag.Get("env"); s != "" {
				name, _, _ := strings.Cut(fld.Tag.Get("env"), ",")
				if name == "-" {
					return ""
				}
				return name
			}
			name, _, _ := strings.Cut(fld.Tag.Get("yaml"), ",")
			if name == "-" {
				return ""
			}
			return name
		})
	}
	return validate
}
