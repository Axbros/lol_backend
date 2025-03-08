package model

import (
	"time"
)

type SmsHistory struct {
	ID       uint64     `gorm:"column:id;type:int(11);primary_key;AUTO_INCREMENT" json:"id"` // 序号
	UserName string     `gorm:"column:user_name;type:varchar(6)" json:"userName"`            // 收信人
	Mobile   string     `gorm:"column:mobile;type:varchar(14)" json:"mobile"`                // 手机号
	CreateAt *time.Time `gorm:"column:create_at;type:datetime" json:"createAt"`              // 创建时间
}

// TableName table name
func (m *SmsHistory) TableName() string {
	return "sms_history"
}
