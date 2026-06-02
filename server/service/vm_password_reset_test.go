package service

import (
	"strings"
	"testing"
)

func TestValidateResetGuestPasswordParamsForWindows(t *testing.T) {
	testCases := []struct {
		name      string
		username  string
		password  string
		wantError string
	}{
		{
			name:     "accept windows administrator",
			username: "administrator",
			password: "ResetAa12345!",
		},
		{
			name:      "reject empty username",
			username:  "",
			password:  "ResetAa12345!",
			wantError: "请输入要重置的用户名",
		},
		{
			name:      "reject unsupported windows username chars",
			username:  "admin/user",
			password:  "ResetAa12345!",
			wantError: "Windows 用户名包含不支持的字符",
		},
		{
			name:      "reject weak password",
			username:  "administrator",
			password:  "weakpass",
			wantError: "密码长度不能少于",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := ValidateResetGuestPasswordParams(testCase.username, testCase.password, "windows")
			if testCase.wantError == "" {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", testCase.wantError)
			}
			if !strings.Contains(err.Error(), testCase.wantError) {
				t.Fatalf("expected error containing %q, got %q", testCase.wantError, err.Error())
			}
		})
	}
}

func TestBuildWindowsPasswordResetScript(t *testing.T) {
	script := buildWindowsPasswordResetScript("administrator", `Reset%^Aa12345!`)

	expectedContains := []string{
		"setlocal DisableDelayedExpansion",
		`reg add "HKLM\SYSTEM\Setup" /v SetupType /t REG_DWORD /d 0 /f >nul 2>&1`,
		`reg add "HKLM\SYSTEM\Setup" /v OOBEInProgress /t REG_DWORD /d 0 /f >nul 2>&1`,
		`net user "administrator" "Reset%%^^Aa12345!"`,
		`shutdown /s /t 5 /f`,
		toWindowsBatchPath(windowsResetDoneGuestPath),
		toWindowsBatchPath(windowsResetErrorGuestPath),
	}

	for _, expected := range expectedContains {
		if !strings.Contains(script, expected) {
			t.Fatalf("expected script to contain %q, got:\n%s", expected, script)
		}
	}
}

func TestBuildWindowsSetupRegFile(t *testing.T) {
	content := buildWindowsSetupRegFile()

	expectedContains := []string{
		"Windows Registry Editor Version 5.00",
		`[HKEY_LOCAL_MACHINE\SYSTEM\Setup]`,
		`"CmdLine"="cmd.exe /c C:\\ProgramData\\kvm-console\\reset-password.cmd"`,
		`"SetupType"=dword:00000002`,
		`"SystemSetupInProgress"=dword:00000001`,
		`"OOBEInProgress"=dword:00000001`,
	}

	for _, expected := range expectedContains {
		if !strings.Contains(content, expected) {
			t.Fatalf("expected reg file to contain %q, got:\n%s", expected, content)
		}
	}
}
