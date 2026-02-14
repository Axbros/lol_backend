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

	overdueDays := 0
	var lastPayDate time.Time
	// 解析还款日期
	returnDateInt, err := strconv.Atoi(loanRecord.LoanReturnDate)
	if err != nil {
		logger.Errorf("failed to convert string to int, string: %s", loanRecord.LoanReturnDate)
		return nil, err
	}

	// 获取每月固定还款日
	dueDay := returnDateInt
	now := time.Now()

	if paidSuccessLength > 0 {
		lastPayRecord := paymentHistoryRecords[paidSuccessLength-1]
		if lastPayRecord.CreateAt == nil {
			logger.Errorf("lastPayRecord.CreateAt is nil for mobile: %s", mobile)
			return nil, errors.New("last payment record create time is nil")
		}
		lastPayDate = *lastPayRecord.CreateAt

		// 计算逾期天数：从应还款日（还款月的dueDay）到最后一次还款日期
		overdueDays, err = calculateCurrentOverdueDays(lastPayDate, dueDay)
		if err != nil {
			logger.Errorf("calculate overdue days failed, mobile: %s, err: %v", mobile, err)
			return nil, err
		}
	} else {
		// 从未还款：计算从最近一个应还款日到当前日期的逾期天数
		overdueDays, err = calculateCurrentOverdueDays(now, dueDay)
		if err != nil {
			logger.Errorf("calculate overdue days failed, mobile: %s, err: %v", mobile, err)
			return nil, err
		}
	}

	// 计算当月应还款日
	shouldPayDate, err := getValidDueDate(now.Year(), now.Month(), dueDay)
	if err != nil {
		logger.Errorf("get valid due date failed, year: %d, month: %d, day: %d, err: %v", now.Year(), now.Month(), dueDay, err)
		return nil, err
	}
	loanRecord.ShouldPayDate = shouldPayDate
	loanRecord.LastPayDate = lastPayDate

	// 判断是否逾期
	isOverdue := false
	if paidSuccessLength > 0 {
		// 有还款记录：判断最后一次还款是否晚于对应还款周期的应还款日
		// 计算最后一次还款所在月的应还款日
		payMonthDueDate, err := getValidDueDate(lastPayDate.Year(), lastPayDate.Month(), dueDay)
		if err != nil {
			return nil, err
		}
		isOverdue = lastPayDate.After(payMonthDueDate)
	} else {
		// 无还款记录：判断当前日期是否晚于当月应还款日
		isOverdue = now.After(shouldPayDate)
	}

	if isOverdue {
		loanRecord.OverDueDays = overdueDays
		loanRecord.OverDueMoney = overdueDays * 100
	} else {
		// 未逾期则清空逾期相关字段
		loanRecord.OverDueDays = 0
		loanRecord.OverDueMoney = 0
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

// calculateCurrentOverdueDays 计算【当前】整体逾期天数（核心解决你的问题）
// dueDay：每月固定还款日（比如27）
// lastPayDate：最后一次还款日期（用于判断该算哪一期的应还款日）
func calculateCurrentOverdueDays(lastPayDate time.Time, dueDay int) (int, error) {
	now := time.Now() // 今天：2026-02-14
	var latestDueDate time.Time

	// ========== 场景1：无还款记录（lastPayDate 是零值） ==========
	if lastPayDate.IsZero() {
		// 从未还款：最近一期应还款日 = 上个月的dueDay（比如2月→1月27日）
		lastMonth := now.Month() - 1
		lastYear := now.Year()
		if lastMonth < 1 {
			lastMonth = 12
			lastYear -= 1
		}
		var err error
		latestDueDate, err = getValidDueDate(lastYear, lastMonth, dueDay)
		if err != nil {
			return 0, err
		}
		// ========== 场景2：有还款记录 ==========
	} else {
		// 1. 先算「最后还款月」的应还款日
		payYear, payMonth := lastPayDate.Year(), lastPayDate.Month()
		payMonthDueDate, err := getValidDueDate(payYear, payMonth, dueDay)
		if err != nil {
			return 0, err
		}

		// 2. 判断最后还款是否晚于「还款月的应还款日」
		if lastPayDate.After(payMonthDueDate) {
			// 例子：lastPayDate=12.29 > 12.27 → 最近一期应还款日=1.27
			nextMonth := payMonth + 1
			nextYear := payYear
			if nextMonth > 12 {
				nextMonth = 1
				nextYear += 1
			}
			latestDueDate, err = getValidDueDate(nextYear, nextMonth, dueDay)
		} else {
			// 还款在应还款日之前 → 最近一期就是还款月的应还款日
			latestDueDate = payMonthDueDate
		}
	}

	// 第二步：如果当前时间没到最近一期应还款日 → 0天逾期
	if now.Before(latestDueDate) {
		return 0, nil
	}

	// 第三步：计算当前逾期天数（精确到天）
	diff := now.Sub(latestDueDate)
	overdueDays := int(diff.Hours() / 24)

	return overdueDays, nil
}

// calculateOverdueDaysNoPayment 计算无还款记录时的逾期天数
// now: 当前日期
// dueDay: 每月固定还款日
func calculateOverdueDaysNoPayment(now time.Time, dueDay int) (int, error) {
	// 计算当月应还款日
	dueDate, err := getValidDueDate(now.Year(), now.Month(), dueDay)
	if err != nil {
		return 0, err
	}

	// 如果当前日期在应还款日之前/当天，没有逾期
	if !now.After(dueDate) {
		return 0, nil
	}

	// 计算逾期天数
	diff := now.Sub(dueDate)
	overdueDays := int(diff.Hours() / 24)

	return overdueDays, nil
}

// getPaymentHistory 查询支付成功的历史记录
func (d *loanDao) getPaymentHistory(ctx context.Context, mobile string) ([]*model.PaymentHistory, error) {
	var paymentHistoryRecords []*model.PaymentHistory
	if err := d.db.Model(&model.PaymentHistory{}).WithContext(ctx).Where("user_phone = ? AND status = 'SUCCESS'", mobile).Find(&paymentHistoryRecords).Error; err != nil {
		return nil, err
	}
	return paymentHistoryRecords, nil
}

func (d *loanDao) CreatePaymentHistory(ctx context.Context, table *model.PaymentHistory) error {
	return d.db.Model(&model.PaymentHistory{}).Create(table).Error
}

func (d *loanDao) UpdatePaymentStatusByTradeNo(ctx context.Context, tradeNo string, status string) error {
	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		err := d.db.Model(&model.PaymentHistory{}).Where("out_trade_no = ?", tradeNo).Update("status", status).Error
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
