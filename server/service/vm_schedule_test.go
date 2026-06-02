package service

import (
	"testing"
	"time"
)

func TestCalculateNextRunOnce(t *testing.T) {
	location := time.FixedZone("UTC+8", 8*3600)
	now := time.Date(2026, 5, 1, 8, 0, 0, 0, location)
	runAt := time.Date(2026, 5, 1, 9, 30, 0, 0, location)

	nextRun, err := calculateNextRun(
		VMScheduleEventTypePower,
		VMScheduleActionStart,
		VMScheduleTypeOnce,
		&runAt,
		"",
		nil,
		location,
		now,
	)
	if err != nil {
		t.Fatalf("calculateNextRun returned error: %v", err)
	}
	if nextRun == nil {
		t.Fatalf("expected nextRun to be set")
	}
	if !nextRun.Equal(runAt) {
		t.Fatalf("expected %v, got %v", runAt, *nextRun)
	}
}

func TestCalculateNextRunDailyMovesToTomorrow(t *testing.T) {
	location := time.FixedZone("UTC+8", 8*3600)
	now := time.Date(2026, 5, 1, 10, 5, 0, 0, location)

	nextRun, err := calculateNextRun(
		VMScheduleEventTypePower,
		VMScheduleActionShutdown,
		VMScheduleTypeDaily,
		nil,
		"10:00",
		nil,
		location,
		now,
	)
	if err != nil {
		t.Fatalf("calculateNextRun returned error: %v", err)
	}
	expected := time.Date(2026, 5, 2, 10, 0, 0, 0, location)
	if nextRun == nil || !nextRun.Equal(expected) {
		t.Fatalf("expected %v, got %v", expected, nextRun)
	}
}

func TestCalculateNextRunWeeklyChoosesNearestWeekday(t *testing.T) {
	location := time.FixedZone("UTC+8", 8*3600)
	now := time.Date(2026, 5, 1, 9, 0, 0, 0, location) // 周五

	nextRun, err := calculateNextRun(
		VMScheduleEventTypePower,
		VMScheduleActionStart,
		VMScheduleTypeWeekly,
		nil,
		"12:00",
		[]int{1, 5},
		location,
		now,
	)
	if err != nil {
		t.Fatalf("calculateNextRun returned error: %v", err)
	}
	expected := time.Date(2026, 5, 1, 12, 0, 0, 0, location)
	if nextRun == nil || !nextRun.Equal(expected) {
		t.Fatalf("expected %v, got %v", expected, nextRun)
	}
}

func TestBuildVMScheduleModelRejectsDeleteRecurring(t *testing.T) {
	enabled := true
	_, err := buildVMScheduleModel(nil, "demo", "admin", VMScheduleInput{
		EventType:    VMScheduleEventTypeLifecycle,
		Action:       VMScheduleActionDelete,
		ScheduleType: VMScheduleTypeDaily,
		TimeOfDay:    "10:00",
		Timezone:     "UTC",
		Enabled:      &enabled,
	})
	if err == nil {
		t.Fatalf("expected error for recurring delete schedule")
	}
}
