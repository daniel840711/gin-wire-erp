package request

import (
	cErr "interchange/internal/pkg/error"
	"regexp"

	"github.com/go-playground/validator/v10"
)

type Validator interface {
	GetMessages() ValidatorMessages
}

type ValidatorMessages map[string]string

var reg = regexp.MustCompile(`\[\d\]`)

// GetError 從請求和錯誤中獲取錯誤信息
func GetError(request interface{}, err error) *cErr.Error {
	if _, isValidatorErrors := err.(validator.ValidationErrors); isValidatorErrors {
		_, isValidator := request.(Validator)

		var errorMessages []string
		for _, v := range err.(validator.ValidationErrors) {
			if isValidator {
				field := v.Field() // 獲取字段名稱
				field = reg.ReplaceAllString(field, ".*")
				if message, exist := request.(Validator).GetMessages()[field+"."+v.Tag()]; exist {
					errorMessages = append(errorMessages, message)
					continue
				}
			}
			errorMessages = append(errorMessages, v.Error())
		}
		if len(errorMessages) > 0 {
			return cErr.ValidateErr(errorMessages[0]) // Return the first error message
		}
	}

	return cErr.ValidateErr("Parameter error")
}
