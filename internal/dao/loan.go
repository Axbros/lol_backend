package dao

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sync/singleflight"
	"gorm.io/gorm"

	"github.com/go-dev-frame/sponge/pkg/logger"
	"github.com/go-dev-frame/sponge/pkg/sgorm/query"
	"github.com/go-dev-frame/sponge/pkg/utils"

	"lol/internal/cache"
	"lol/internal/database"
	"lol/internal/model"
)

var _ LoanDao = (*loanDao)(nil)

// LoanDao defining the dao interface
type LoanDao interface {
	Create(ctx context.Context, table *model.Loan) error
	DeleteByID(ctx context.Context, id uint64) error
	UpdateByID(ctx context.Context, table *model.Loan) error
	GetByID(ctx context.Context, id uint64) (*model.Loan, error)
	GetByColumns(ctx context.Context, params *query.Params) ([]*model.Loan, int64, error)

	CreateByTx(ctx context.Context, tx *gorm.DB, table *model.Loan) (uint64, error)
	DeleteByTx(ctx context.Context, tx *gorm.DB, id uint64) error
	UpdateByTx(ctx context.Context, tx *gorm.DB, table *model.Loan) error
	GetByMobileAndCode(ctx context.Context, mobile string, code string) (*model.Loan, error)
	CreatePaymentHistory(ctx context.Context, table *model.PaymentHistory) error
	GetPaymentHistoryByTradeNo(ctx context.Context, tradeNo string) (*model.PaymentHistory, error)
	UpdatePaymentStatusByTradeNo(ctx context.Context, tradeNo string, status string) error
	getPaymentHistory(ctx context.Context, mobile string) ([]*model.PaymentHistory, error)
}

type loanDao struct {
	db    *gorm.DB
	cache cache.LoanCache     // if nil, the cache is not used.
	sfg   *singleflight.Group // if cache is nil, the sfg is not used.
}

// NewLoanDao creating the dao interface
func NewLoanDao(db *gorm.DB, xCache cache.LoanCache) LoanDao {
	if xCache == nil {
		return &loanDao{db: db}
	}
	return &loanDao{
		db:    db,
		cache: xCache,
		sfg:   new(singleflight.Group),
	}
}

func (d *loanDao) deleteCache(ctx context.Context, id uint64) error {
	if d.cache != nil {
		return d.cache.Del(ctx, id)
	}
	return nil
}

// Create a record, insert the record and the id value is written back to the table
func (d *loanDao) Create(ctx context.Context, table *model.Loan) error {
	return d.db.WithContext(ctx).Create(table).Error
}

// DeleteByID delete a record by id
func (d *loanDao) DeleteByID(ctx context.Context, id uint64) error {
	err := d.db.WithContext(ctx).Where("id = ?", id).Delete(&model.Loan{}).Error
	if err != nil {
		return err
	}

	// delete cache
	_ = d.deleteCache(ctx, id)

	return nil
}

// UpdateByID update a record by id
func (d *loanDao) UpdateByID(ctx context.Context, table *model.Loan) error {
	err := d.updateDataByID(ctx, d.db, table)

	// delete cache
	_ = d.deleteCache(ctx, table.ID)

	return err
}

func (d *loanDao) updateDataByID(ctx context.Context, db *gorm.DB, table *model.Loan) error {
	if table.ID < 1 {
		return errors.New("id cannot be 0")
	}

	update := map[string]interface{}{}

	if table.Name != "" {
		update["name"] = table.Name
	}
	if table.UserID != "" {
		update["user_id"] = table.UserID
	}
	if table.Mobile != "" {
		update["mobile"] = table.Mobile
	}
	if table.CarModel != "" {
		update["car_model"] = table.CarModel
	}
	if table.CarPlate != "" {
		update["car_plate"] = table.CarPlate
	}
	if table.LoanMoney != 0 {
		update["loan_money"] = table.LoanMoney
	}
	if table.LoanPeriod != 0 {
		update["loan_period"] = table.LoanPeriod
	}
	if table.LoanReturnDate != "" {
		update["loan_return_date"] = table.LoanReturnDate
	}
	if table.MonthlyPayment != 0 {
		update["monthly_payment"] = table.MonthlyPayment
	}
	if table.CreateAt.IsZero() == false {
		update["create_at"] = table.CreateAt
	}
	if table.Status != 0 {
		update["status"] = table.Status
	}

	return db.WithContext(ctx).Model(table).Updates(update).Error
}

// GetByID get a record by id
func (d *loanDao) GetByID(ctx context.Context, id uint64) (*model.Loan, error) {
	// no cache
	if d.cache == nil {
		record := &model.Loan{}
		err := d.db.WithContext(ctx).Where("id = ?", id).First(record).Error
		return record, err
	}

	// get from cache
	record, err := d.cache.Get(ctx, id)
	if err == nil {
		return record, nil
	}

	// get from database
	if errors.Is(err, database.ErrCacheNotFound) {
		// for the same id, prevent high concurrent simultaneous access to database
		val, err, _ := d.sfg.Do(utils.Uint64ToStr(id), func() (interface{}, error) { //nolint
			table := &model.Loan{}
			err = d.db.WithContext(ctx).Where("id = ?", id).First(table).Error
			if err != nil {
				if errors.Is(err, database.ErrRecordNotFound) {
					// set placeholder cache to prevent cache penetration, default expiration time 10 minutes
					if err = d.cache.SetPlaceholder(ctx, id); err != nil {
						logger.Warn("cache.SetPlaceholder error", logger.Err(err), logger.Any("id", id))
					}
					return nil, database.ErrRecordNotFound
				}
				return nil, err
			}
			// set cache
			if err = d.cache.Set(ctx, id, table, cache.LoanExpireTime); err != nil {
				logger.Warn("cache.Set error", logger.Err(err), logger.Any("id", id))
			}
			return table, nil
		})
		if err != nil {
			return nil, err
		}
		table, ok := val.(*model.Loan)
		if !ok {
			return nil, database.ErrRecordNotFound
		}
		return table, nil
	}

	if d.cache.IsPlaceholderErr(err) {
		return nil, database.ErrRecordNotFound
	}

	return nil, err
}

// GetByColumns get paging records by column information,
// Note: query performance degrades when table rows are very large because of the use of offset.
//
// params includes paging parameters and query parameters
// paging parameters (required):
//
//	page: page number, starting from 0
//	limit: lines per page
//	sort: sort fields, default is id backwards, you can add - sign before the field to indicate reverse order, no - sign to indicate ascending order, multiple fields separated by comma
//
// query parameters (not required):
//
//	name: column name
//	exp: expressions, which default is "=",  support =, !=, >, >=, <, <=, like, in, notin, isnull, isnotnull
//	value: column value, if exp=in, multiple values are separated by commas
//	logic: logical type, default value is "and", support &, and, ||, or
//
// example: search for a male over 20 years of age
//
//	params = &query.Params{
//	    Page: 0,
//	    Limit: 20,
//	    Columns: []query.Column{
//		{
//			Name:    "age",
//			Exp: ">",
//			Value:   20,
//		},
//		{
//			Name:  "gender",
//			Value: "male",
//		},
//	}
func (d *loanDao) GetByColumns(ctx context.Context, params *query.Params) ([]*model.Loan, int64, error) {
	queryStr, args, err := params.ConvertToGormConditions()
	if err != nil {
		return nil, 0, errors.New("query params error: " + err.Error())
	}

	var total int64
	if params.Sort != "ignore count" { // determine if count is required
		err = d.db.WithContext(ctx).Model(&model.Loan{}).Where(queryStr, args...).Count(&total).Error
		if err != nil {
			return nil, 0, err
		}
		if total == 0 {
			return nil, total, nil
		}
	}

	records := []*model.Loan{}
	order, limit, offset := params.ConvertToPage()
	err = d.db.WithContext(ctx).Order(order).Limit(limit).Offset(offset).Where(queryStr, args...).Find(&records).Error
	if err != nil {
		return nil, 0, err
	}

	return records, total, err
}

// CreateByTx create a record in the database using the provided transaction
func (d *loanDao) CreateByTx(ctx context.Context, tx *gorm.DB, table *model.Loan) (uint64, error) {
	err := tx.WithContext(ctx).Create(table).Error
	return table.ID, err
}

// DeleteByTx delete a record by id in the database using the provided transaction
func (d *loanDao) DeleteByTx(ctx context.Context, tx *gorm.DB, id uint64) error {
	err := tx.WithContext(ctx).Where("id = ?", id).Delete(&model.Loan{}).Error
	if err != nil {
		return err
	}

	// delete cache
	_ = d.deleteCache(ctx, id)

	return nil
}

// UpdateByTx update a record by id in the database using the provided transaction
func (d *loanDao) UpdateByTx(ctx context.Context, tx *gorm.DB, table *model.Loan) error {
	err := d.updateDataByID(ctx, tx, table)

	// delete cache
	_ = d.deleteCache(ctx, table.ID)

	return err
}

// GetByMobileAndCode 根据手机号和代码获取贷款记录并计算逾期信息
func (d *loanDao) GetByMobileAndCode(ctx context.Context, mobile string, code string) (*model.Loan, error) {
	// 查询贷款记录
	loanRecord := &model.Loan{}
	if err := d.db.WithContext(ctx).Where("mobile = ? AND RIGHT(user_id, 6) = ?", mobile, code).First(loanRecord).Error; err != nil {
		return nil, err
	}
	if loanRecord.Status == 1 {
		//表示已经处理完毕 无需用户还款了
		return loanRecord, nil
	}

	// 查询支付成功的历史记录
	paymentHistoryRecords, err := d.getPaymentHistory(ctx, mobile)
	if err != nil {
		return nil, err
	}

	// 处理支付成功记录
	paidSuccessLength := 0
	for _, paymentHistoryRecord := range paymentHistoryRecords {
		if paymentHistoryRecord.Status == "SUCCESS" {
			paidSuccessLength++
		}
	}
	loanRecord.PaidCount = paidSuccessLength

	var lastPayDate time.Time
	// 解析还款日期
	returnDateInt, err := strconv.Atoi(loanRecord.LoanReturnDate)
	if err != nil {
		logger.Errorf("failed to convert string to int, string: %s", loanRecord.LoanReturnDate)
		return nil, err
	}

	dueDay := returnDateInt
	now := time.Now()
	if loanRecord.CreateAt == nil {
		return nil, errors.New("loan create time is nil")
	}
	if paidSuccessLength > 0 {
		lastPayRecord := paymentHistoryRecords[paidSuccessLength-1]
		if lastPayRecord.CreateAt == nil {
			return nil, errors.New("last payment record create time is nil")
		}
		lastPayDate = *lastPayRecord.CreateAt
	}

	// 首期还款日取注册日期之后最近的还款日；每成功支付一期，顺延一个月。
	shouldPayDate, err := calculateNextPaymentDueDate(*loanRecord.CreateAt, paidSuccessLength, dueDay)
	if err != nil {
		logger.Errorf("calculate next payment due date failed, mobile: %s, err: %v", mobile, err)
		return nil, err
	}
	loanRecord.ShouldPayDate = shouldPayDate
	loanRecord.LastPayDate = lastPayDate

	nowDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	if nowDate.After(shouldPayDate) {
		loanRecord.OverDueDays = int(nowDate.Sub(shouldPayDate).Hours() / 24)
		loanRecord.OverDueMoney = loanRecord.OverDueDays * 100
	}

	return loanRecord, nil
}

// getValidDueDate 计算有效还款日（处理当月天数不足的情况，比如2月没有31号）
func getValidDueDate(year int, month time.Month, dueDay int) (time.Time, error) {
	// 获取当月最后一天
	lastDay := getLastDayOfMonth(year, month)
	if dueDay > lastDay {
		// 如果还款日大于当月最后一天，取当月最后一天
		return time.Date(year, month, lastDay, 0, 0, 0, 0, time.Local), nil
	}
	return time.Date(year, month, dueDay, 0, 0, 0, 0, time.Local), nil
}

// getLastDayOfMonth 获取指定年月的最后一天
func getLastDayOfMonth(year int, month time.Month) int {
	// 下个月第一天减一天就是当月最后一天
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.Local).Day()
}

func calculateNextPaymentDueDate(createAt time.Time, paidCount int, dueDay int) (time.Time, error) {
	if dueDay < 1 || dueDay > 31 {
		return time.Time{}, errors.New("payment due day must be between 1 and 31")
	}
	if paidCount < 0 {
		return time.Time{}, errors.New("paid count cannot be negative")
	}

	createDate := time.Date(createAt.Year(), createAt.Month(), createAt.Day(), 0, 0, 0, 0, time.Local)
	firstDueDate, err := getValidDueDate(createAt.Year(), createAt.Month(), dueDay)
	if err != nil {
		return time.Time{}, err
	}
	monthOffset := 0
	if !createDate.Before(firstDueDate) {
		monthOffset = 1
	}

	dueMonth := time.Date(createAt.Year(), createAt.Month()+time.Month(monthOffset+paidCount), 1, 0, 0, 0, 0, time.Local)
	return getValidDueDate(dueMonth.Year(), dueMonth.Month(), dueDay)
}

// getPaymentHistory 查询支付成功的历史记录
func (d *loanDao) getPaymentHistory(ctx context.Context, mobile string) ([]*model.PaymentHistory, error) {
	var paymentHistoryRecords []*model.PaymentHistory
	if err := d.db.WithContext(ctx).Where("user_phone = ? AND status = 'SUCCESS'", mobile).Order("create_at ASC").Find(&paymentHistoryRecords).Error; err != nil {
		return nil, err
	}
	return paymentHistoryRecords, nil
}

func (d *loanDao) CreatePaymentHistory(ctx context.Context, table *model.PaymentHistory) error {
	return d.db.WithContext(ctx).Create(table).Error
}

func (d *loanDao) GetPaymentHistoryByTradeNo(ctx context.Context, tradeNo string) (*model.PaymentHistory, error) {
	record := &model.PaymentHistory{}
	err := d.db.WithContext(ctx).Where("out_trade_no = ?", tradeNo).First(record).Error
	return record, err
}

func (d *loanDao) UpdatePaymentStatusByTradeNo(ctx context.Context, tradeNo string, status string) error {
	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		err := d.db.WithContext(ctx).Model(&model.PaymentHistory{}).Where("out_trade_no = ?", tradeNo).Update("status", status).Error
		if err == nil {
			return nil
		}
		if !isConnectionError(err) {
			return err
		}
		// 打印重试信息
		fmt.Printf("第 %d 次更新支付状态失败，原因: %v，将在 2 秒后重试...\n", i+1, err)
		// 等待一段时间后重试
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("更新支付状态失败，经过 %d 次重试后仍然失败", maxRetries)
}

// isConnectionError 检查错误是否是连接相关的错误
func isConnectionError(err error) bool {
	// 这里可以根据具体的错误信息进行判断
	errorMessages := []string{
		"read tcp",
		"connection reset by peer",
		"broken pipe",
		"i/o timeout",
	}
	for _, msg := range errorMessages {
		if strings.Contains(err.Error(), msg) {
			return true
		}
	}
	return false
}
