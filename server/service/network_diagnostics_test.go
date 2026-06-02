package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"kvm_console/config"
)

func TestBuildNetworkCaptureBPF(t *testing.T) {
	tests := []struct {
		name    string
		filter  NetworkDiagnosticFilter
		want    string
		wantErr string
	}{
		{
			name:   "tcp with host and port",
			filter: NetworkDiagnosticFilter{Protocol: "tcp", SourceIP: "192.168.122.10", DestPort: 22},
			want:   "tcp and src host 192.168.122.10 and dst port 22",
		},
		{
			name:   "dhcp template",
			filter: NetworkDiagnosticFilter{Protocol: "dhcp"},
			want:   "(udp and (port 67 or port 68))",
		},
		{
			name:   "dns template",
			filter: NetworkDiagnosticFilter{Protocol: "dns"},
			want:   "port 53",
		},
		{
			name:    "invalid source ip",
			filter:  NetworkDiagnosticFilter{SourceIP: "not-an-ip"},
			wantErr: "源 IP 格式不正确",
		},
		{
			name:    "invalid port",
			filter:  NetworkDiagnosticFilter{Protocol: "udp", Port: 70000},
			wantErr: "端口范围必须为 1-65535",
		},
		{
			name:    "arp with port",
			filter:  NetworkDiagnosticFilter{Protocol: "arp", Port: 80},
			wantErr: "ARP 协议不能同时指定端口过滤",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := BuildNetworkCaptureBPF(tt.filter)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestNetworkCaptureSessionDownloadPath(t *testing.T) {
	tmpDir := t.TempDir()
	oldConfig := config.GlobalConfig
	config.GlobalConfig = &config.Config{NetworkCaptureDir: tmpDir}
	defer func() { config.GlobalConfig = oldConfig }()

	taskID := uint(42)
	defer func() {
		captureSessionsMu.Lock()
		delete(captureSessions, taskID)
		captureSessionsMu.Unlock()
	}()
	fileName := "capture-42-test.pcap"
	filePath := filepath.Join(tmpDir, fileName)
	if err := os.WriteFile(filePath, []byte("pcap"), 0o600); err != nil {
		t.Fatalf("write capture file: %v", err)
	}

	InitNetworkCaptureSession(taskID, "vm-a", NetworkCaptureRequest{}, "admin")
	updateNetworkCaptureSession(taskID, func(session *NetworkCaptureSession) {
		session.FileName = fileName
		session.DownloadPath = "/api/network/captures/42/download"
		session.Status = "success"
	})

	gotPath, gotName, err := NetworkCaptureFilePath(taskID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotPath != filePath || gotName != fileName {
		t.Fatalf("unexpected file path/name: %s %s", gotPath, gotName)
	}
}

func TestDeleteNetworkCaptureFileClearsSession(t *testing.T) {
	tmpDir := t.TempDir()
	oldConfig := config.GlobalConfig
	config.GlobalConfig = &config.Config{NetworkCaptureDir: tmpDir}
	defer func() { config.GlobalConfig = oldConfig }()

	taskID := uint(43)
	defer func() {
		captureSessionsMu.Lock()
		delete(captureSessions, taskID)
		captureSessionsMu.Unlock()
	}()
	fileName := "capture-43-test.pcap"
	filePath := filepath.Join(tmpDir, fileName)
	if err := os.WriteFile(filePath, []byte("pcap"), 0o600); err != nil {
		t.Fatalf("write capture file: %v", err)
	}

	InitNetworkCaptureSession(taskID, "vm-a", NetworkCaptureRequest{}, "admin")
	updateNetworkCaptureSession(taskID, func(session *NetworkCaptureSession) {
		session.FileName = fileName
		session.DownloadPath = "/api/network/captures/43/download"
		session.FileSize = 4
		session.Status = "success"
	})

	if err := DeleteNetworkCaptureFile(taskID); err != nil {
		t.Fatalf("delete capture file: %v", err)
	}
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Fatalf("expected capture file removed, stat err=%v", err)
	}
	session, ok := GetNetworkCaptureSession(taskID)
	if !ok {
		t.Fatalf("expected session to remain")
	}
	if session.FileName != "" || session.DownloadPath != "" || session.FileSize != 0 {
		t.Fatalf("expected file fields cleared, got %+v", session)
	}
}

func TestInitNetworkCaptureSessionDeletesPreviousFile(t *testing.T) {
	tmpDir := t.TempDir()
	oldConfig := config.GlobalConfig
	config.GlobalConfig = &config.Config{NetworkCaptureDir: tmpDir}
	defer func() { config.GlobalConfig = oldConfig }()

	oldTaskID := uint(44)
	newTaskID := uint(45)
	defer func() {
		captureSessionsMu.Lock()
		delete(captureSessions, oldTaskID)
		delete(captureSessions, newTaskID)
		captureSessionsMu.Unlock()
	}()
	fileName := "capture-44-test.pcap"
	filePath := filepath.Join(tmpDir, fileName)
	if err := os.WriteFile(filePath, []byte("pcap"), 0o600); err != nil {
		t.Fatalf("write capture file: %v", err)
	}

	InitNetworkCaptureSession(oldTaskID, "vm-a", NetworkCaptureRequest{}, "admin")
	updateNetworkCaptureSession(oldTaskID, func(session *NetworkCaptureSession) {
		session.FileName = fileName
		session.DownloadPath = "/api/network/captures/44/download"
		session.FileSize = 4
		session.Status = "success"
	})
	InitNetworkCaptureSession(newTaskID, "vm-b", NetworkCaptureRequest{}, "admin")

	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Fatalf("expected previous capture file removed, stat err=%v", err)
	}
	session, ok := GetNetworkCaptureSession(oldTaskID)
	if !ok {
		t.Fatalf("expected old session to remain")
	}
	if session.FileName != "" || session.DownloadPath != "" || session.FileSize != 0 {
		t.Fatalf("expected old session file fields cleared, got %+v", session)
	}
}

func TestClampNetworkCaptureLimits(t *testing.T) {
	if got := clampInt(0, 30, 120); got != 30 {
		t.Fatalf("expected default 30, got %d", got)
	}
	if got := clampInt(300, 30, 120); got != 120 {
		t.Fatalf("expected max 120, got %d", got)
	}
	if got := clampInt(60, 30, 120); got != 60 {
		t.Fatalf("expected 60, got %d", got)
	}
}
