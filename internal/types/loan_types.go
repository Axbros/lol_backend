package types

import (
	"time"

	"github.com/go-dev-frame/sponge/pkg/sgorm/query"
)

var _ time.Time

// Tip: suggested filling in the binding rules https://github.com/go-playground/validator in request struct fields tag.

// CreateLoanRequest request params
type CreateLoanRequest struct {
	Name           string `json:"name" binding:""`           // 姓名
	UserID         string `json:"userID" binding:""`         // 身份证号码
	Mobile         string `json:"mobile" binding:""`         // 手机号码
	CarModel       string `json:"carModel" binding:""`       // 车型
	CarPlate       string `json:"carPlate" binding:""`       // 车牌
	LoanMoney      int    `json:"loanMoney" binding:""`      // 借款金额
	LoanPeriod     int    `json:"loanPeriod" binding:""`     // 借款期数
	LoanReturnDate string `json:"loanReturnDate" binding:""` // 还款日期
}

type GetDetailRequest struct {
	Mobile string `json:"mobile" binding:""` // mobile id
	Code   string `json:"code" binding:""`   // code
}

type PayRequest struct {
	Mobile string `json:"mobile" binding:""` // mobile id
	Code   string `json:"code" binding:""`   // code
	Method string `json:"method" binding:""` // method
}

// UpdateLoanByIDRequest request params
type UpdateLoanByIDRequest struct {
	ID uint64 `json:"id" binding:""` // uint64 id
	// 序号
	Name           string     `json:"name" binding:""`           // 姓名
	UserID         string     `json:"userID" binding:""`         // 身份证号码
	Mobile         string     `json:"mobile" binding:""`         // 手机号码
	CarModel       string     `json:"carModel" binding:""`       // 车型
	CarPlate       string     `json:"carPlate" binding:""`       // 车牌
	LoanMoney      float64    `json:"loanMoney" binding:""`      // 借款金额
	LoanPeriod     int        `json:"loanPeriod" binding:""`     // 借款期数
	LoanReturnDate string     `json:"loanReturnDate" binding:""` // 还款日期
	MonthlyPayment int        `json:"monthlyPayment" binding:""` // 每月应还
	CreateAt       *time.Time `json:"createAt" binding:""`       // 创建时间
	Status         string     `json:"status" binding:""`         // 状态
}

// LoanObjDetail detail
type LoanObjDetail struct {
	ID uint64 `json:"id"` // convert to uint64 id
	// 序号
	Name           string     `json:"name"`           // 姓名
	UserID         string     `json:"userID"`         // 身份证号码
	Mobile         string     `json:"mobile"`         // 手机号码
	CarModel       string     `json:"carModel"`       // 车型
	CarPlate       string     `json:"carPlate"`       // 车牌
	LoanMoney      float64    `json:"loanMoney"`      // 借款金额
	LoanPeriod     int        `json:"loanPeriod"`     // 借款期数
	LoanReturnDate string     `json:"loanReturnDate"` // 还款日期
	MonthlyPayment int        `json:"monthlyPayment"` // 每月应还
	CreateAt       *time.Time `json:"createAt"`       // 创建时间
	Status         string     `json:"status"`         // 状态
}

// CreateLoanReply only for api docs
type CreateLoanReply struct {
	Code int    `json:"code"` // return code
	Msg  string `json:"msg"`  // return information description
	Data struct {
		ID uint64 `json:"id"` // id
	} `json:"data"` // return data
}

// DeleteLoanByIDReply only for api docs
type DeleteLoanByIDReply struct {
	Result
}

// UpdateLoanByIDReply only for api docs
type UpdateLoanByIDReply struct {
	Result
}

// GetLoanByIDReply only for api docs
type GetLoanByIDReply struct {
	Code int    `json:"code"` // return code
	Msg  string `json:"msg"`  // return information description
	Data struct {
		Loan LoanObjDetail `json:"loan"`
	} `json:"data"` // return data
}

// ListLoansRequest request params
type ListLoansRequest struct {
	query.Params
}

// ListLoansReply only for api docs
type ListLoansReply struct {
	Code int    `json:"code"` // return code
	Msg  string `json:"msg"`  // return information description
	Data struct {
		Loans []LoanObjDetail `json:"loans"`
	} `json:"data"` // return data
}
