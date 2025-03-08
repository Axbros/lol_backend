package model

import (
	"time"
)

type Loan struct {
	ID             uint64     `gorm:"column:id;type:int(11);primary_key;AUTO_INCREMENT" json:"id"`   // 序号
	Name           string     `gorm:"column:name;type:varchar(10)" json:"name"`                      // 姓名
	UserID         string     `gorm:"column:user_id;type:varchar(18)" json:"userID"`                 // 身份证号码
	Mobile         string     `gorm:"column:mobile;type:varchar(11)" json:"mobile"`                  // 手机号码
	CarModel       string     `gorm:"column:car_model;type:varchar(15)" json:"carModel"`             // 车型
	CarPlate       string     `gorm:"column:car_plate;type:varchar(10)" json:"carPlate"`             // 车牌
	LoanMoney      float64    `gorm:"column:loan_money;type:double" json:"loanMoney"`                // 借款金额
	LoanPeriod     int        `gorm:"column:loan_period;type:int(11)" json:"loanPeriod"`             // 借款期数
	LoanReturnDate string     `gorm:"column:loan_return_date;type:varchar(2)" json:"loanReturnDate"` // 还款日期
	MonthlyPayment float64    `gorm:"column:monthly_payment;type:double" json:"monthlyPayment"`      // 每月应还
	CreateAt       *time.Time `gorm:"column:create_at;type:datetime" json:"createAt"`                // 创建时间
	Status         int        `gorm:"column:status;type:tinyint" json:"status"`                      // 状态
	PaidCount      int64      `gorm:"column:paid_count;type:int(11)" json:"paidCount"`               // 已还期数
}

// TableName table name
func (m *Loan) TableName() string {
	return "loan"
}
