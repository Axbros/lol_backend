package types

import (
	"time"

	"github.com/go-dev-frame/sponge/pkg/sgorm/query"
)

var _ time.Time

// Tip: suggested filling in the binding rules https://github.com/go-playground/validator in request struct fields tag.

// CreateSmsHistoryRequest request params
type CreateSmsHistoryRequest struct {
	UserName string     `json:"userName" binding:""` // 收信人
	Mobile   string     `json:"mobile" binding:""`   // 手机号
	CreateAt *time.Time `json:"createAt" binding:""` // 创建时间
}

// UpdateSmsHistoryByIDRequest request params
type UpdateSmsHistoryByIDRequest struct {
	ID uint64 `json:"id" binding:""` // uint64 id
	// 序号
	UserName string     `json:"userName" binding:""` // 收信人
	Mobile   string     `json:"mobile" binding:""`   // 手机号
	CreateAt *time.Time `json:"createAt" binding:""` // 创建时间
}

// SmsHistoryObjDetail detail
type SmsHistoryObjDetail struct {
	ID uint64 `json:"id"` // convert to uint64 id
	// 序号
	UserName string     `json:"userName"` // 收信人
	Mobile   string     `json:"mobile"`   // 手机号
	CreateAt *time.Time `json:"createAt"` // 创建时间
}

// CreateSmsHistoryReply only for api docs
type CreateSmsHistoryReply struct {
	Code int    `json:"code"` // return code
	Msg  string `json:"msg"`  // return information description
	Data struct {
		ID uint64 `json:"id"` // id
	} `json:"data"` // return data
}

// DeleteSmsHistoryByIDReply only for api docs
type DeleteSmsHistoryByIDReply struct {
	Result
}

// UpdateSmsHistoryByIDReply only for api docs
type UpdateSmsHistoryByIDReply struct {
	Result
}

// GetSmsHistoryByIDReply only for api docs
type GetSmsHistoryByIDReply struct {
	Code int    `json:"code"` // return code
	Msg  string `json:"msg"`  // return information description
	Data struct {
		SmsHistory SmsHistoryObjDetail `json:"smsHistory"`
	} `json:"data"` // return data
}

// ListSmsHistorysRequest request params
type ListSmsHistorysRequest struct {
	query.Params
}

// ListSmsHistorysReply only for api docs
type ListSmsHistorysReply struct {
	Code int    `json:"code"` // return code
	Msg  string `json:"msg"`  // return information description
	Data struct {
		SmsHistorys []SmsHistoryObjDetail `json:"smsHistorys"`
	} `json:"data"` // return data
}
