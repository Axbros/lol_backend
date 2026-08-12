package dao

import (
	"testing"
	"time"
)

func TestCalculateNextPaymentDueDate(t *testing.T) {
	tests := []struct {
		name      string
		createdAt time.Time
		paidCount int
		dueDay    int
		want      time.Time
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
			name:      "one installment already paid",
			createdAt: time.Date(2026, time.June, 13, 10, 0, 0, 0, time.Local),
			paidCount: 1,
			dueDay:    12,
			want:      time.Date(2026, time.August, 12, 0, 0, 0, 0, time.Local),
		},
		{
			name:      "month with fewer days",
			createdAt: time.Date(2026, time.January, 15, 10, 0, 0, 0, time.Local),
			paidCount: 1,
			dueDay:    31,
			want:      time.Date(2026, time.February, 28, 0, 0, 0, 0, time.Local),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := calculateNextPaymentDueDate(tt.createdAt, tt.paidCount, tt.dueDay)
			if err != nil {
				t.Fatal(err)
			}
			if !got.Equal(tt.want) {
				t.Fatalf("due date = %v, want %v", got, tt.want)
			}
		})
	}
}
