package ecode

import (
	"github.com/go-dev-frame/sponge/pkg/errcode"
)

// result business-level http error codes.
// the resultNO value range is 1~100, if the same error code is used, it will cause panic.
var (
	resultNO       = 37
	resultName     = "result"
	resultBaseCode = errcode.HCode(resultNO)

	ErrCreateResult     = errcode.NewError(resultBaseCode+1, "failed to create "+resultName)
	ErrDeleteByIDResult = errcode.NewError(resultBaseCode+2, "failed to delete "+resultName)
	ErrUpdateByIDResult = errcode.NewError(resultBaseCode+3, "failed to update "+resultName)
	ErrGetByIDResult    = errcode.NewError(resultBaseCode+4, "failed to get "+resultName+" details")
	ErrListResult       = errcode.NewError(resultBaseCode+5, "failed to list of "+resultName)

	// error codes are globally unique, adding 1 to the previous error code
)
