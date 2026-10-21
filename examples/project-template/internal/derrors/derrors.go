package derrors

import (
	"fmt"
	"net/http"
)

type DetailedError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details"`

	err error `json:"-"`
}

func (me *DetailedError) SetCode(code int) *DetailedError {
	me.Code = code
	return me
}

func (me *DetailedError) SetMessage(message string) *DetailedError {
	me.Message = message
	return me
}

func (me *DetailedError) SetDetails(details any) *DetailedError {
	me.Details = details
	return me
}

func (me *DetailedError) SetError(err error) *DetailedError {
	me.err = err
	return me
}

func NewInternalError(err error) *DetailedError {
	return &DetailedError{
		Code:    http.StatusInternalServerError * 1000,
		Message: "系统内部错误",
		err:     err,
	}
}

func NewUnAuthError(err error) *DetailedError {
	return &DetailedError{
		Code:    http.StatusUnauthorized * 1000,
		Message: "未授权",
		err:     err,
	}
}

func NewParamError(err error) *DetailedError {
	return &DetailedError{
		Code:    http.StatusBadRequest * 1000,
		Message: "请求参数错误",
		err:     err,
	}
}

func NewNotFoundError(err error) *DetailedError {
	return &DetailedError{
		Code:    http.StatusNotFound * 1000,
		Message: "未找到",
		err:     err,
	}
}

func NewForbiddenError(err error) *DetailedError {
	return &DetailedError{
		Code:    http.StatusForbidden * 1000,
		Message: "禁止访问",
		err:     err,
	}
}

func (me *DetailedError) Error() string {
	if me.err != nil {
		return me.err.Error()
	}
	return fmt.Sprintf("%v:%v", me.Code, me.Message)
}
