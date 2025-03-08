package model

import (
	"time"
)

type PaymentHistory struct {
	ID         uint64     `gorm:"column:id;type:int(11);primary_key;AUTO_INCREMENT" json:"id"` // 序号
	UserPhone  string     `gorm:"column:user_phone;type:varchar(11)" json:"userPhone"`         // 用户手机号码
	OutTradeNo string     `gorm:"column:out_trade_no;type:varchar(255)" json:"outTradeNo"`     // 支付订单号
	Status     string     `gorm:"column:status;type:varchar(12)" json:"status"`                // 状态
	Method     string     `gorm:"column:method;type:varchar(12)" json:"method"`                // 支付方式
	CreateAt   *time.Time `gorm:"column:create_at;type:datetime" json:"createAt"`              // 创建时间
}

// TableName table name
func (m *PaymentHistory) TableName() string {
	return "payment_history"
}
