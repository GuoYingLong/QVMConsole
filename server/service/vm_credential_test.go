package service

import (
	"strings"
	"testing"

	"kvm_console/config"
)

func TestEncryptAndDecryptVMSecret(t *testing.T) {
	config.GlobalConfig = &config.Config{
		VMCredentialSecret: "unit-test-secret",
	}

	plainText := "ResetAa12345!"
	encrypted, err := encryptVMSecret(plainText)
	if err != nil {
		t.Fatalf("encryptVMSecret returned error: %v", err)
	}
	if encrypted == plainText {
		t.Fatalf("expected encrypted text to differ from plain text")
	}

	decrypted, err := decryptVMSecret(encrypted)
	if err != nil {
		t.Fatalf("decryptVMSecret returned error: %v", err)
	}
	if decrypted != plainText {
		t.Fatalf("expected decrypted text %q, got %q", plainText, decrypted)
	}
}

func TestValidateResetLinuxPasswordParams(t *testing.T) {
	testCases := []struct {
		name      string
		username  string
		password  string
		wantError string
	}{
		{
			name:     "accept valid params",
			username: "xiaozhu",
			password: "ResetAa12345!",
		},
		{
			name:      "reject empty username",
			username:  "",
			password:  "ResetAa12345!",
			wantError: "请输入要重置的用户名",
		},
		{
			name:      "reject invalid username",
			username:  "User.Name",
			password:  "ResetAa12345!",
			wantError: "用户名只能以小写字母或下划线开头",
		},
		{
			name:      "reject weak password",
			username:  "xiaozhu",
			password:  "weakpass",
			wantError: "密码长度不能少于",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := ValidateResetLinuxPasswordParams(testCase.username, testCase.password)
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
