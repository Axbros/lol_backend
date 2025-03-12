package dao

import (
	"context"
	"errors"
	"fmt"
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

func (d *loanDao) GetByMobileAndCode(ctx context.Context, mobile string, code string) (*model.Loan, error) {
	record := &model.Loan{}
	err := d.db.WithContext(ctx).Where("mobile = ? AND RIGHT(user_id, 6) = ?", mobile, code).First(record).Error
	if err != nil {
		return nil, err
	}
	err = d.db.Model(&model.PaymentHistory{}).Where("user_phone = ? AND status = 'SUCCESS'", mobile).Count(&record.PaidCount).Error
	return record, err
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
