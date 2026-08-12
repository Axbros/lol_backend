package reminder

import (
	"testing"
	"time"

	"lol/internal/model"
)

func TestIsReminderDay(t *testing.T) {
	loc := time.FixedZone("CST", 8*60*60)
	createdBefore := time.Date(2026, time.July, 13, 10, 0, 0, 0, loc)
	createdToday := time.Date(2026, time.August, 12, 8, 0, 0, 0, loc)
	tests := []struct {
		name string
		user *model.Loan
		now  time.Time
		want bool
	}{
		{"day before due", &model.Loan{CreateAt: &createdBefore, LoanReturnDate: "12"}, time.Date(2026, time.August, 11, 9, 0, 0, 0, loc), true},
		{"due day", &model.Loan{CreateAt: &createdBefore, LoanReturnDate: "12"}, time.Date(2026, time.August, 12, 9, 0, 0, 0, loc), true},
		{"registration day is not due", &model.Loan{CreateAt: &createdToday, LoanReturnDate: "12"}, time.Date(2026, time.August, 12, 9, 0, 0, 0, loc), false},
		{"two days before due", &model.Loan{CreateAt: &createdBefore, LoanReturnDate: "12"}, time.Date(2026, time.August, 10, 9, 0, 0, 0, loc), false},
		{"completed loan", &model.Loan{CreateAt: &createdBefore, LoanReturnDate: "12", Status: 1}, time.Date(2026, time.August, 12, 9, 0, 0, 0, loc), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isReminderDay(tt.user, tt.now); got != tt.want {
				t.Fatalf("isReminderDay() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsReminderDayUsesLastDayOfShortMonth(t *testing.T) {
	loc := time.FixedZone("CST", 8*60*60)
	created := time.Date(2026, time.January, 1, 0, 0, 0, 0, loc)
	user := &model.Loan{CreateAt: &created, LoanReturnDate: "31"}
	if !isReminderDay(user, time.Date(2026, time.February, 28, 9, 0, 0, 0, loc)) {
		t.Fatal("expected February 28 to be due for day 31")
	}
}

func TestIsReminderDayHandlesMonthBoundary(t *testing.T) {
	loc := time.FixedZone("CST", 8*60*60)
	created := time.Date(2026, time.January, 1, 0, 0, 0, 0, loc)
	user := &model.Loan{CreateAt: &created, LoanReturnDate: "1"}
	if !isReminderDay(user, time.Date(2026, time.August, 31, 9, 0, 0, 0, loc)) {
		t.Fatal("expected August 31 to be the advance reminder for September 1")
	}
}

func TestNextRunAt(t *testing.T) {
	now := time.Date(2026, time.August, 12, 10, 0, 0, 0, time.Local)
	want := time.Date(2026, time.August, 13, 9, 0, 0, 0, time.Local)
	if got := nextRunAt(now, 9, 0); !got.Equal(want) {
		t.Fatalf("nextRunAt() = %v, want %v", got, want)
	}
}
