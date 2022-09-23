package api

import (
	"reflect"
	"strings"

	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	en_translations "github.com/go-playground/validator/v10/translations/en"
)

const (
	jsonTagName  = "json"
	emptyTagName = "-"
	subString    = 2
)

// configureValidator configure the struct validator
func configureValidator(validate *validator.Validate) error {
	eng := en.New()
	uni := ut.New(eng, eng)
	trans, _ := uni.GetTranslator("en")
	if err := en_translations.RegisterDefaultTranslations(validate, trans); err != nil {
		return err
	}
	validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get(jsonTagName), ",", subString)[0]
		if name == emptyTagName {
			return fld.Name
		}
		return name
	})
	return nil
}
