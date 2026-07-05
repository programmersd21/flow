package history

import (
	"testing"
	"time"
)

func TestTracker_Record_SameDay(t *testing.T) {
	// Mock time for a single day: 2026-07-05
	mockTime := time.Date(2026, 7, 5, 10, 0, 0, 0, time.UTC)
	nowFunc = func() time.Time { return mockTime }
	defer func() { nowFunc = time.Now }()

	tracker := NewTracker()
	if tracker.Year != 2026 || tracker.Month != 7 || tracker.Day != 5 {
		t.Errorf("expected tracker initialized to 2026-07-05, got %d-%d-%d", tracker.Year, tracker.Month, tracker.Day)
	}

	tracker.Record(100, 50, 1.0)
	tracker.Record(200, 100, 1.0)

	if tracker.TodayDown != 300 {
		t.Errorf("expected TodayDown 300, got %f", tracker.TodayDown)
	}
	if tracker.TodayUp != 150 {
		t.Errorf("expected TodayUp 150, got %f", tracker.TodayUp)
	}
	if tracker.PeakDown != 200 {
		t.Errorf("expected PeakDown 200, got %f", tracker.PeakDown)
	}
	if tracker.PeakUp != 100 {
		t.Errorf("expected PeakUp 100, got %f", tracker.PeakUp)
	}
}

func TestTracker_Record_MonthChange_SameDayNumber(t *testing.T) {
	// Mock time start: Jan 15
	mockTime := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
	nowFunc = func() time.Time { return mockTime }
	defer func() { nowFunc = time.Now }()

	tracker := NewTracker()
	tracker.Record(100, 50, 1.0)

	if tracker.TodayDown != 100 {
		t.Errorf("expected TodayDown 100, got %f", tracker.TodayDown)
	}

	// Change time to Feb 15 (month change, same day number)
	mockTime = time.Date(2026, 2, 15, 12, 0, 0, 0, time.UTC)
	tracker.Record(200, 80, 1.0)

	if tracker.TodayDown != 200 {
		t.Errorf("expected TodayDown to reset to 200 on month change, got %f", tracker.TodayDown)
	}
	if tracker.TodayUp != 80 {
		t.Errorf("expected TodayUp to reset to 80 on month change, got %f", tracker.TodayUp)
	}
	// Peaks should be retained across day changes
	if tracker.PeakDown != 200 {
		t.Errorf("expected PeakDown 200, got %f", tracker.PeakDown)
	}
}

func TestTracker_Record_YearChange(t *testing.T) {
	// Mock time start: Dec 31
	mockTime := time.Date(2025, 12, 31, 23, 59, 0, 0, time.UTC)
	nowFunc = func() time.Time { return mockTime }
	defer func() { nowFunc = time.Now }()

	tracker := NewTracker()
	tracker.Record(100, 50, 1.0)

	// Change time to Jan 1 of next year
	mockTime = time.Date(2026, 1, 1, 0, 1, 0, 0, time.UTC)
	tracker.Record(300, 150, 2.0)

	if tracker.TodayDown != 600 {
		t.Errorf("expected TodayDown to reset and be 600, got %f", tracker.TodayDown)
	}
	if tracker.TodayUp != 300 {
		t.Errorf("expected TodayUp to reset and be 300, got %f", tracker.TodayUp)
	}
}
