package types

import (
	"time"

	"github.com/go-dev-frame/sponge/pkg/sgorm/query"
)

var _ time.Time

// Tip: suggested filling in the binding rules https://github.com/go-playground/validator in request struct fields tag.

// CreateResultRequest request params
type CreateResultRequest struct {
	EventType      string     `json:"eventType" binding:""`      // 事件类型
	ResourceAppid  string     `json:"resourceAppid" binding:""`  // 来源ID
	ResourceMchid  string     `json:"resourceMchid" binding:""`  // 来源商户ID
	OutTradeNo     string     `json:"outTradeNo" binding:""`     // 订单号
	TransactionID  string     `json:"transactionID" binding:""`  // 交易ID
	TradeType      string     `json:"tradeType" binding:""`      // 交易类型
	TradeState     string     `json:"tradeState" binding:""`     // 交易状态
	TradeStateDesc string     `json:"tradeStateDesc" binding:""` // 交易状态描述
	BankType       string     `json:"bankType" binding:""`       // 银行类型
	Attach         string     `json:"attach" binding:""`
	SuccessTime    string     `json:"successTime" binding:""` // 成功时间
	Payer          string     `json:"payer" binding:""`       // 支付人
	AmountTotal    float64    `json:"amountTotal" binding:""` // 合计
	CreateAt       *time.Time `json:"createAt" binding:""`    // 创建时间
}

// UpdateResultByIDRequest request params
type UpdateResultByIDRequest struct {
	ID uint64 `json:"id" binding:""` // uint64 id
	// 序号
	EventType      string     `json:"eventType" binding:""`      // 事件类型
	ResourceAppid  string     `json:"resourceAppid" binding:""`  // 来源ID
	ResourceMchid  string     `json:"resourceMchid" binding:""`  // 来源商户ID
	OutTradeNo     string     `json:"outTradeNo" binding:""`     // 订单号
	TransactionID  string     `json:"transactionID" binding:""`  // 交易ID
	TradeType      string     `json:"tradeType" binding:""`      // 交易类型
	TradeState     string     `json:"tradeState" binding:""`     // 交易状态
	TradeStateDesc string     `json:"tradeStateDesc" binding:""` // 交易状态描述
	BankType       string     `json:"bankType" binding:""`       // 银行类型
	Attach         string     `json:"attach" binding:""`
	SuccessTime    string     `json:"successTime" binding:""` // 成功时间
	Payer          string     `json:"payer" binding:""`       // 支付人
	AmountTotal    float64    `json:"amountTotal" binding:""` // 合计
	CreateAt       *time.Time `json:"createAt" binding:""`    // 创建时间
}

// ResultObjDetail detail
type ResultObjDetail struct {
	ID uint64 `json:"id"` // convert to uint64 id
	// 序号
	EventType      string     `json:"eventType"`      // 事件类型
	ResourceAppid  string     `json:"resourceAppid"`  // 来源ID
	ResourceMchid  string     `json:"resourceMchid"`  // 来源商户ID
	OutTradeNo     string     `json:"outTradeNo"`     // 订单号
	TransactionID  string     `json:"transactionID"`  // 交易ID
	TradeType      string     `json:"tradeType"`      // 交易类型
	TradeState     string     `json:"tradeState"`     // 交易状态
	TradeStateDesc string     `json:"tradeStateDesc"` // 交易状态描述
	BankType       string     `json:"bankType"`       // 银行类型
	Attach         string     `json:"attach"`
	SuccessTime    string     `json:"successTime"` // 成功时间
	Payer          string     `json:"payer"`       // 支付人
	AmountTotal    float64    `json:"amountTotal"` // 合计
	CreateAt       *time.Time `json:"createAt"`    // 创建时间
}

// CreateResultReply only for api docs
type CreateResultReply struct {
	Code int    `json:"code"` // return code
	Msg  string `json:"msg"`  // return information description
	Data struct {
		ID uint64 `json:"id"` // id
	} `json:"data"` // return data
}

// DeleteResultByIDReply only for api docs
type DeleteResultByIDReply struct {
	Result
}

// UpdateResultByIDReply only for api docs
type UpdateResultByIDReply struct {
	Result
}

// GetResultByIDReply only for api docs
type GetResultByIDReply struct {
	Code int    `json:"code"` // return code
	Msg  string `json:"msg"`  // return information description
	Data struct {
		Result ResultObjDetail `json:"result"`
	} `json:"data"` // return data
}

// ListResultsRequest request params
type ListResultsRequest struct {
	query.Params
}

// ListResultsReply only for api docs
type ListResultsReply struct {
	Code int    `json:"code"` // return code
	Msg  string `json:"msg"`  // return information description
	Data struct {
		Results []ResultObjDetail `json:"results"`
	} `json:"data"` // return data
}
