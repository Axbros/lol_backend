package ecode

import (
	"github.com/go-dev-frame/sponge/pkg/errcode"
)

// smsHistory business-level http error codes.
// the smsHistoryNO value range is 1~100, if the same error code is used, it will cause panic.
var (
	smsHistoryNO       = 79
	smsHistoryName     = "smsHistory"
	smsHistoryBaseCode = errcode.HCode(smsHistoryNO)

	ErrCreateSmsHistory     = errcode.NewError(smsHistoryBaseCode+1, "failed to create "+smsHistoryName)
	ErrDeleteByIDSmsHistory = errcode.NewError(smsHistoryBaseCode+2, "failed to delete "+smsHistoryName)
	ErrUpdateByIDSmsHistory = errcode.NewError(smsHistoryBaseCode+3, "failed to update "+smsHistoryName)
	ErrGetByIDSmsHistory    = errcode.NewError(smsHistoryBaseCode+4, "failed to get "+smsHistoryName+" details")
	ErrListSmsHistory       = errcode.NewError(smsHistoryBaseCode+5, "failed to list of "+smsHistoryName)

	// error codes are globally unique, adding 1 to the previous error code
)
