package dao

import (
	"testing"
	"time"
)

func TestCalculateNextPaymentDueDate(t *testing.T) {
	tests := []struct {
		name        string
		createdAt   time.Time
		lastPayDate time.Time
		dueDay      int
		want        time.Time
	}{
		{
			name:      "registered after this month's due date",
			createdAt: time.Date(2026, time.June, 13, 10, 0, 0, 0, time.Local),
			dueDay:    12,
			want:      time.Date(2026, time.July, 12, 0, 0, 0, 0, time.Local),
		},
		{
			name:      "registered before this month's due date",
			createdAt: time.Date(2026, time.June, 10, 10, 0, 0, 0, time.Local),
			dueDay:    12,
			want:      time.Date(2026, time.June, 12, 0, 0, 0, 0, time.Local),
		},
		{
			name:        "one installment already paid",
			createdAt:   time.Date(2026, time.June, 13, 10, 0, 0, 0, time.Local),
			lastPayDate: time.Date(2026, time.July, 12, 10, 0, 0, 0, time.Local),
			dueDay:      12,
			want:        time.Date(2026, time.August, 12, 0, 0, 0, 0, time.Local),
		},
		{
			name:        "legacy user uses latest payment instead of registration count",
			createdAt:   time.Date(2025, time.January, 13, 10, 0, 0, 0, time.Local),
			lastPayDate: time.Date(2026, time.July, 20, 10, 0, 0, 0, time.Local),
			dueDay:      12,
			want:        time.Date(2026, time.August, 12, 0, 0, 0, 0, time.Local),
		},
		{
			name:        "month with fewer days",
			createdAt:   time.Date(2025, time.January, 15, 10, 0, 0, 0, time.Local),
			lastPayDate: time.Date(2026, time.January, 31, 10, 0, 0, 0, time.Local),
			dueDay:      31,
			want:        time.Date(2026, time.February, 28, 0, 0, 0, 0, time.Local),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := calculateNextPaymentDueDate(tt.createdAt, tt.lastPayDate, tt.dueDay)
			if err != nil {
				t.Fatal(err)
			}
			if !got.Equal(tt.want) {
				t.Fatalf("due date = %v, want %v", got, tt.want)
			}
		})
	}
}
