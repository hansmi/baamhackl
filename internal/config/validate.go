package config

import (
	"reflect"
	"strings"
	"sync"

	"github.com/go-playground/validator/v10"
)

var customValidate customValidateImpl

type customValidateImpl struct {
	once sync.Once
	v    *validator.Validate
}

func (c *customValidateImpl) get() *validator.Validate {
	c.once.Do(func() {
		c.v = validator.New()
		c.v.RegisterTagNameFunc(func(fld reflect.StructField) string {
			name := strings.SplitN(fld.Tag.Get("yaml"), ",", 2)[0]
			if name == "-" {
				return ""
			}
			return name
		})
	})
	return c.v
}
