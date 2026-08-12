package reminder

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/go-dev-frame/sponge/pkg/logger"
	"gorm.io/gorm"

	"lol/internal/config"
	"lol/internal/model"
	"lol/internal/sms"
)

const schedulerLockName = "lol:sms-reminder"

type Service struct {
	db     *gorm.DB
	sender sms.Sender
	cfg    config.SMS
	loc    *time.Location
	stop   chan struct{}
	done   chan struct{}
	once   sync.Once
}

func NewService(db *gorm.DB, sender sms.Sender, cfg config.SMS) (*Service, error) {
	if db == nil || sender == nil {
		return nil, errors.New("SMS reminder dependencies cannot be nil")
	}
	if cfg.Hour < 0 || cfg.Hour > 23 || cfg.Minute < 0 || cfg.Minute > 59 {
		return nil, errors.New("SMS reminder schedule is invalid")
	}
	timezone := cfg.Timezone
	if timezone == "" {
		timezone = "Asia/Shanghai"
	}
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return nil, fmt.Errorf("load SMS timezone: %w", err)
	}
	if cfg.MaxRetries <= 0 {
		cfg.MaxRetries = 5
	}
	if cfg.RetryDelay <= 0 {
		cfg.RetryDelay = 5
	}
	if cfg.TaskTimeout <= 0 {
		cfg.TaskTimeout = 300
	}
	return &Service{db: db, sender: sender, cfg: cfg, loc: loc, stop: make(chan struct{}), done: make(chan struct{})}, nil
}

func (s *Service) String() string { return "sms reminder scheduler" }

func (s *Service) Start() error {
	defer close(s.done)
	logger.Infof("SMS reminder scheduler started, schedule=%02d:%02d timezone=%s", s.cfg.Hour, s.cfg.Minute, s.loc.String())
	if s.cfg.RunOnStart {
		s.run()
	}

	for {
		now := time.Now().In(s.loc)
		next := nextRunAt(now, s.cfg.Hour, s.cfg.Minute)
		timer := time.NewTimer(time.Until(next))
		select {
		case <-timer.C:
			s.run()
		case <-s.stop:
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			return nil
		}
	}
}

func (s *Service) Stop() error {
	s.once.Do(func() { close(s.stop) })
	<-s.done
	return nil
}

func (s *Service) run() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(s.cfg.TaskTimeout)*time.Second)
	defer cancel()
	if err := s.waitForDatabase(ctx); err != nil {
		logger.Error("SMS reminder database unavailable", logger.Err(err))
		return
	}

	conn, acquired, err := s.acquireLock(ctx)
	if err != nil {
		logger.Error("acquire SMS reminder lock failed", logger.Err(err))
		return
	}
	if !acquired {
		logger.Info("SMS reminder skipped because another instance is running")
		return
	}
	defer func() {
		releaseCtx, releaseCancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer releaseCancel()
		if _, releaseErr := conn.ExecContext(releaseCtx, "SELECT RELEASE_LOCK(?)", schedulerLockName); releaseErr != nil {
			logger.Error("release SMS reminder lock failed", logger.Err(releaseErr))
		}
		_ = conn.Close()
	}()

	if err := s.process(ctx, time.Now().In(s.loc)); err != nil {
		logger.Error("SMS reminder task failed", logger.Err(err))
	}
}

func (s *Service) waitForDatabase(ctx context.Context) error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	for attempt := 1; attempt <= s.cfg.MaxRetries; attempt++ {
		if err = sqlDB.PingContext(ctx); err == nil {
			return nil
		}
		logger.Errorf("database connection failed (%d/%d): %v", attempt, s.cfg.MaxRetries, err)
		if attempt == s.cfg.MaxRetries {
			break
		}
		select {
		case <-time.After(time.Duration(s.cfg.RetryDelay) * time.Second):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return fmt.Errorf("database unavailable after %d attempts: %w", s.cfg.MaxRetries, err)
}

func (s *Service) acquireLock(ctx context.Context) (*sql.Conn, bool, error) {
	sqlDB, err := s.db.DB()
	if err != nil {
		return nil, false, err
	}
	conn, err := sqlDB.Conn(ctx)
	if err != nil {
		return nil, false, err
	}
	var acquired sql.NullInt64
	if err := conn.QueryRowContext(ctx, "SELECT GET_LOCK(?, 0)", schedulerLockName).Scan(&acquired); err != nil {
		_ = conn.Close()
		return nil, false, err
	}
	if !acquired.Valid || acquired.Int64 != 1 {
		_ = conn.Close()
		return nil, false, nil
	}
	return conn, true, nil
}

func (s *Service) process(ctx context.Context, now time.Time) error {
	var users []*model.Loan
	if err := s.db.WithContext(ctx).Where("status IS NULL OR status <> ?", 1).Find(&users).Error; err != nil {
		return err
	}

	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, s.loc)
	nextDay := dayStart.AddDate(0, 0, 1)
	for _, user := range users {
		if err := ctx.Err(); err != nil {
			return err
		}
		if !isReminderDay(user, now) {
			continue
		}
		mobile := sms.FormatMobile(user.Mobile, s.cfg.CountryCode)
		var count int64
		if err := s.db.WithContext(ctx).Model(&model.SmsHistory{}).
			Where("mobile = ? AND create_at >= ? AND create_at < ?", mobile, dayStart, nextDay).
			Count(&count).Error; err != nil {
			logger.Error("query SMS history failed", logger.Err(err), logger.String("mobile", mobile))
			continue
		}
		if count > 0 {
			logger.Infof("SMS reminder already sent today, user=%s mobile=%s", user.Name, mobile)
			continue
		}
		if len(user.UserID) < 6 {
			logger.Warnf("SMS reminder skipped because user ID is too short, user=%s", user.Name)
			continue
		}
		code := user.UserID[len(user.UserID)-6:]
		if err := s.sender.Send(ctx, mobile, user.Name, code); err != nil {
			logger.Error("send SMS reminder failed", logger.Err(err), logger.String("user", user.Name), logger.String("mobile", mobile))
			continue
		}
		sentAt := now
		if err := s.db.WithContext(ctx).Create(&model.SmsHistory{UserName: user.Name, Mobile: mobile, CreateAt: &sentAt}).Error; err != nil {
			logger.Error("save SMS history failed", logger.Err(err), logger.String("user", user.Name), logger.String("mobile", mobile))
			continue
		}
		logger.Infof("SMS reminder sent successfully, user=%s mobile=%s", user.Name, mobile)
	}
	return nil
}

func isReminderDay(user *model.Loan, now time.Time) bool {
	if user == nil || user.Status == 1 || user.CreateAt == nil || user.LoanReturnDate == "" {
		return false
	}
	dueDay, err := strconv.Atoi(user.LoanReturnDate)
	if err != nil || dueDay < 1 || dueDay > 31 {
		return false
	}
	created := user.CreateAt.In(now.Location())
	createDate := time.Date(created.Year(), created.Month(), created.Day(), 0, 0, 0, 0, now.Location())
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// Check both this month's and next month's due dates. The latter covers a
	// due date on the first day of a month, whose advance reminder is in the
	// previous month.
	for _, monthOffset := range []int{0, 1} {
		month := time.Date(now.Year(), now.Month()+time.Month(monthOffset), 1, 0, 0, 0, 0, now.Location())
		dueDate := validDueDate(month.Year(), month.Month(), dueDay, now.Location())
		if !createDate.Before(dueDate) {
			continue
		}
		if today.Equal(dueDate) || today.Equal(dueDate.AddDate(0, 0, -1)) {
			return true
		}
	}
	return false
}

func validDueDate(year int, month time.Month, dueDay int, loc *time.Location) time.Time {
	lastDay := time.Date(year, month+1, 0, 0, 0, 0, 0, loc).Day()
	if dueDay > lastDay {
		dueDay = lastDay
	}
	return time.Date(year, month, dueDay, 0, 0, 0, 0, loc)
}

func nextRunAt(now time.Time, hour int, minute int) time.Time {
	next := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())
	if !next.After(now) {
		next = next.AddDate(0, 0, 1)
	}
	return next
}
