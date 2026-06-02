package service

import (
	"testing"
	"time"
)

func TestApplyVMRuntimeObservation(t *testing.T) {
	observedAt := time.Date(2026, 4, 29, 10, 0, 0, 0, time.UTC)
	record := &vmRuntimeRecord{VMName: "vm-test"}

	if changed := applyVMRuntimeObservation(record, "running", observedAt); !changed {
		t.Fatalf("expected first running observation to change record")
	}
	if record.CurrentRunStartedAt == nil {
		t.Fatalf("expected CurrentRunStartedAt to be set")
	}
	if !record.CurrentRunStartedAt.Equal(observedAt) {
		t.Fatalf("unexpected started time: %v", record.CurrentRunStartedAt)
	}
	if record.LastState != "running" {
		t.Fatalf("unexpected state: %s", record.LastState)
	}

	if changed := applyVMRuntimeObservation(record, "running", observedAt.Add(30*time.Second)); changed {
		t.Fatalf("expected repeated running observation not to change record")
	}

	if changed := applyVMRuntimeObservation(record, "shut off", observedAt.Add(2*time.Minute)); !changed {
		t.Fatalf("expected shutdown observation to change record")
	}
	if record.CurrentRunStartedAt != nil {
		t.Fatalf("expected CurrentRunStartedAt to be cleared after shutdown")
	}
	if record.LastState != "shut off" {
		t.Fatalf("unexpected state after shutdown: %s", record.LastState)
	}
}

func TestBuildVMRuntimeInfo(t *testing.T) {
	startedAt := time.Date(2026, 4, 29, 11, 0, 0, 0, time.UTC)
	now := startedAt.Add(2*time.Hour + 5*time.Minute + 12*time.Second)
	record := &vmRuntimeRecord{
		VMName:              "vm-test",
		LastState:           "running",
		CurrentRunStartedAt: &startedAt,
	}

	info := buildVMRuntimeInfo(record, "running", now)
	if info.ContinuousRuntimeSeconds != int64((2*time.Hour + 5*time.Minute + 12*time.Second).Seconds()) {
		t.Fatalf("unexpected runtime seconds: %d", info.ContinuousRuntimeSeconds)
	}
	if info.ContinuousRunningSince != "2026-04-29 11:00:00" {
		t.Fatalf("unexpected running since: %s", info.ContinuousRunningSince)
	}

	stoppedInfo := buildVMRuntimeInfo(record, "shut off", now)
	if stoppedInfo.ContinuousRuntimeSeconds != 0 || stoppedInfo.ContinuousRunningSince != "" {
		t.Fatalf("expected stopped VM runtime info to be empty, got %+v", stoppedInfo)
	}
}
