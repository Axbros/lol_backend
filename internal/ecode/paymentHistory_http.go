package ecode

import (
	"github.com/go-dev-frame/sponge/pkg/errcode"
)

// paymentHistory business-level http error codes.
// the paymentHistoryNO value range is 1~100, if the same error code is used, it will cause panic.
var (
	paymentHistoryNO       = 45
	paymentHistoryName     = "paymentHistory"
	paymentHistoryBaseCode = errcode.HCode(paymentHistoryNO)

	ErrCreatePaymentHistory     = errcode.NewError(paymentHistoryBaseCode+1, "failed to create "+paymentHistoryName)
	ErrDeleteByIDPaymentHistory = errcode.NewError(paymentHistoryBaseCode+2, "failed to delete "+paymentHistoryName)
	ErrUpdateByIDPaymentHistory = errcode.NewError(paymentHistoryBaseCode+3, "failed to update "+paymentHistoryName)
	ErrGetByIDPaymentHistory    = errcode.NewError(paymentHistoryBaseCode+4, "failed to get "+paymentHistoryName+" details")
	ErrListPaymentHistory       = errcode.NewError(paymentHistoryBaseCode+5, "failed to list of "+paymentHistoryName)

	// error codes are globally unique, adding 1 to the previous error code
)
