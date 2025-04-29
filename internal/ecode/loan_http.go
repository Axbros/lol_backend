package ecode

import (
	"github.com/go-dev-frame/sponge/pkg/errcode"
)

// loan business-level http error codes.
// the loanNO value range is 1~100, if the same error code is used, it will cause panic.
var (
	loanNO       = 34
	loanName     = "loan"
	loanBaseCode = errcode.HCode(loanNO)

	ErrCreateLoan     = errcode.NewError(loanBaseCode+1, "failed to create "+loanName)
	ErrDeleteByIDLoan = errcode.NewError(loanBaseCode+2, "failed to delete "+loanName)
	ErrUpdateByIDLoan = errcode.NewError(loanBaseCode+3, "failed to update "+loanName)
	ErrGetByIDLoan    = errcode.NewError(loanBaseCode+4, "failed to get "+loanName+" details")
	ErrListLoan       = errcode.NewError(loanBaseCode+5, "failed to list of "+loanName+",maybe username or password is wrong!")
	ErrLoanStatus     = errcode.NewError(loanBaseCode+6, "loan status error")
	ErrCreatePayment  = errcode.NewError(loanBaseCode+7, "failed to create payment")

	// error codes are globally unique, adding 1 to the previous error code
)
