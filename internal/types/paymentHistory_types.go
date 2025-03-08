package types

import (
	"time"

	"github.com/go-dev-frame/sponge/pkg/sgorm/query"
)

var _ time.Time

// Tip: suggested filling in the binding rules https://github.com/go-playground/validator in request struct fields tag.

// CreatePaymentHistoryRequest request params
type CreatePaymentHistoryRequest struct {
	UserPhone  string     `json:"userPhone" binding:""`  // 用户手机号码
	OutTradeNo string     `json:"outTradeNo" binding:""` // 支付订单号
	Status     string     `json:"status" binding:""`     // 状态
	CreateAt   *time.Time `json:"createAt" binding:""`   // 创建时间
}

// UpdatePaymentHistoryByIDRequest request params
type UpdatePaymentHistoryByIDRequest struct {
	ID uint64 `json:"id" binding:""` // uint64 id
	// 序号
	UserPhone  string     `json:"userPhone" binding:""`  // 用户手机号码
	OutTradeNo string     `json:"outTradeNo" binding:""` // 支付订单号
	Status     string     `json:"status" binding:""`     // 状态
	CreateAt   *time.Time `json:"createAt" binding:""`   // 创建时间
}

// PaymentHistoryObjDetail detail
type PaymentHistoryObjDetail struct {
	ID uint64 `json:"id"` // convert to uint64 id
	// 序号
	UserPhone  string     `json:"userPhone"`  // 用户手机号码
	OutTradeNo string     `json:"outTradeNo"` // 支付订单号
	Status     string     `json:"status"`     // 状态
	CreateAt   *time.Time `json:"createAt"`   // 创建时间
}

// CreatePaymentHistoryReply only for api docs
type CreatePaymentHistoryReply struct {
	Code int    `json:"code"` // return code
	Msg  string `json:"msg"`  // return information description
	Data struct {
		ID uint64 `json:"id"` // id
	} `json:"data"` // return data
}

// DeletePaymentHistoryByIDReply only for api docs
type DeletePaymentHistoryByIDReply struct {
	Result
}

// UpdatePaymentHistoryByIDReply only for api docs
type UpdatePaymentHistoryByIDReply struct {
	Result
}

// GetPaymentHistoryByIDReply only for api docs
type GetPaymentHistoryByIDReply struct {
	Code int    `json:"code"` // return code
	Msg  string `json:"msg"`  // return information description
	Data struct {
		PaymentHistory PaymentHistoryObjDetail `json:"paymentHistory"`
	} `json:"data"` // return data
}

// ListPaymentHistorysRequest request params
type ListPaymentHistorysRequest struct {
	query.Params
}

// ListPaymentHistorysReply only for api docs
type ListPaymentHistorysReply struct {
	Code int    `json:"code"` // return code
	Msg  string `json:"msg"`  // return information description
	Data struct {
		PaymentHistorys []PaymentHistoryObjDetail `json:"paymentHistorys"`
	} `json:"data"` // return data
}
