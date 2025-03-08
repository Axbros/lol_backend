package model

import (
	"time"
)

type Result struct {
	ID             uint64     `gorm:"column:id;type:int(11);primary_key;AUTO_INCREMENT" json:"id"`     // 序号
	EventType      string     `gorm:"column:event_type;type:varchar(255)" json:"eventType"`            // 事件类型
	ResourceAppid  string     `gorm:"column:resource_appid;type:varchar(255)" json:"resourceAppid"`    // 来源ID
	ResourceMchid  string     `gorm:"column:resource_mchid;type:varchar(255)" json:"resourceMchid"`    // 来源商户ID
	OutTradeNo     string     `gorm:"column:out_trade_no;type:varchar(255)" json:"outTradeNo"`         // 订单号
	TransactionID  string     `gorm:"column:transaction_id;type:varchar(255)" json:"transactionID"`    // 交易ID
	TradeType      string     `gorm:"column:trade_type;type:varchar(255)" json:"tradeType"`            // 交易类型
	TradeState     string     `gorm:"column:trade_state;type:varchar(255)" json:"tradeState"`          // 交易状态
	TradeStateDesc string     `gorm:"column:trade_state_desc;type:varchar(255)" json:"tradeStateDesc"` // 交易状态描述
	BankType       string     `gorm:"column:bank_type;type:varchar(255)" json:"bankType"`              // 银行类型
	Attach         string     `gorm:"column:attach;type:varchar(255)" json:"attach"`
	SuccessTime    string     `gorm:"column:success_time;type:varchar(255)" json:"successTime"`                 // 成功时间
	Payer          string     `gorm:"column:payer;type:varchar(255)" json:"payer"`                              // 支付人
	AmountTotal    float64    `gorm:"column:amount_total;type:float" json:"amountTotal"`                        // 合计
	CreateAt       *time.Time `gorm:"column:create_at;type:datetime;default:CURRENT_TIMESTAMP" json:"createAt"` // 创建时间
}

// TableName table name
func (m *Result) TableName() string {
	return "result"
}
