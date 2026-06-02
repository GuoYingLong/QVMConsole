//go:build !windows

package utils

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestExecShellWithTimeoutKillsChildProcessGroup(t *testing.T) {
	pidFile, err := os.CreateTemp("", "kvm-console-child-*.pid")
	if err != nil {
		t.Fatalf("CreateTemp() error = %v", err)
	}
	pidPath := pidFile.Name()
	_ = pidFile.Close()
	_ = os.Remove(pidPath)
	defer os.Remove(pidPath)

	result := ExecShellWithTimeout(fmt.Sprintf("sleep 30 & echo $! > %s; wait", shellQuoteForTest(pidPath)), 500*time.Millisecond)
	if result.Error == nil {
		t.Fatalf("ExecShellWithTimeout() expected timeout error")
	}

	data, err := os.ReadFile(pidPath)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", pidPath, err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		t.Fatalf("invalid child pid: %q", string(data))
	}
	defer syscall.Kill(pid, syscall.SIGKILL)

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if !processExists(pid) {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("child process %d still exists after command timeout", pid)
}

func TestExecShellContextWithTimeoutKillsChildProcessGroup(t *testing.T) {
	pidFile, err := os.CreateTemp("", "kvm-console-child-*.pid")
	if err != nil {
		t.Fatalf("CreateTemp() error = %v", err)
	}
	pidPath := pidFile.Name()
	_ = pidFile.Close()
	_ = os.Remove(pidPath)
	defer os.Remove(pidPath)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan *CmdResult, 1)
	go func() {
		done <- ExecShellContextWithTimeout(ctx, fmt.Sprintf("sleep 30 & echo $! > %s; wait", shellQuoteForTest(pidPath)), 30*time.Second)
	}()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(pidPath); err == nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	cancel()

	select {
	case result := <-done:
		if result.Error == nil {
			t.Fatalf("ExecShellContextWithTimeout() expected cancel error")
		}
	case <-time.After(3 * time.Second):
		t.Fatalf("ExecShellContextWithTimeout() did not return after cancel")
	}

	data, err := os.ReadFile(pidPath)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", pidPath, err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		t.Fatalf("invalid child pid: %q", string(data))
	}
	defer syscall.Kill(pid, syscall.SIGKILL)

	deadline = time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if !processExists(pid) {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("child process %d still exists after command cancel", pid)
}

func processExists(pid int) bool {
	err := syscall.Kill(pid, 0)
	return err == nil || err == syscall.EPERM
}

func shellQuoteForTest(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}
