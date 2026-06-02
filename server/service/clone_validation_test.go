package service

import (
	"regexp"
	"strings"
	"testing"
)

func TestValidateStrongPassword(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{
			name:     "accept strong password",
			password: "Aa23456789!@",
			wantErr:  false,
		},
		{
			name:     "reject short password",
			password: "Aa12!xyz",
			wantErr:  true,
		},
		{
			name:     "reject password without symbol",
			password: "Aa2345678901",
			wantErr:  true,
		},
		{
			name:     "reject unsupported characters",
			password: "Aa23456789()",
			wantErr:  true,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateStrongPassword(testCase.password)
			if testCase.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !testCase.wantErr && err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}
		})
	}
}

func TestValidateCloneCredentials(t *testing.T) {
	t.Parallel()

	if err := ValidateCloneCredentials("vm-abcd1234", "demo_user", "Aa23456789!@", true); err != nil {
		t.Fatalf("expected clone credentials to be valid, got %v", err)
	}

	if err := ValidateCloneCredentials("vm-abcd1234", "", "Aa23456789!@", true); err == nil {
		t.Fatalf("expected empty username to be rejected")
	}

	if err := ValidateCloneCredentials("invalid_host-", "demo_user", "Aa23456789!@", true); err == nil {
		t.Fatalf("expected invalid hostname to be rejected")
	}
}

func TestNormalizeCloneUsernameForTemplate(t *testing.T) {
	t.Parallel()

	if got := NormalizeCloneUsernameForTemplate("windows", ""); got != "administrator" {
		t.Fatalf("expected windows username default to administrator, got %q", got)
	}

	if got := NormalizeCloneUsernameForTemplate("linux", "demo_user"); got != "demo_user" {
		t.Fatalf("expected linux username to remain unchanged, got %q", got)
	}
}

func TestValidateCloneCredentialsForTemplate(t *testing.T) {
	t.Parallel()

	if err := ValidateCloneCredentialsForTemplate("windows", "vm-abcd1234", "", "Aa23456789!@", true); err != nil {
		t.Fatalf("expected windows credentials to be valid, got %v", err)
	}

	if err := ValidateCloneCredentialsForTemplate("windows", "vm-abcd1234", "demo_user", "Aa23456789!@", true); err == nil {
		t.Fatalf("expected windows username changes to be rejected")
	}

	if err := ValidateCloneCredentialsForTemplate("linux", "vm-abcd1234", "demo_user", "Aa23456789!@", true); err != nil {
		t.Fatalf("expected linux credentials to be valid, got %v", err)
	}
}

func TestGenerateRandomCloneHostname(t *testing.T) {
	t.Parallel()

	hostname := GenerateRandomCloneHostname()
	if !regexp.MustCompile(`^vm-[a-z0-9]{8}$`).MatchString(hostname) {
		t.Fatalf("unexpected hostname format: %s", hostname)
	}
}

func TestNormalizeRequestedDiskSize(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		requested int
		min       int
		want      int
	}{
		{
			name:      "use template size when request is zero",
			requested: 0,
			min:       40,
			want:      40,
		},
		{
			name:      "raise smaller request to template size",
			requested: 20,
			min:       40,
			want:      40,
		},
		{
			name:      "keep larger request",
			requested: 60,
			min:       40,
			want:      60,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if got := NormalizeRequestedDiskSize(testCase.requested, testCase.min); got != testCase.want {
				t.Fatalf("expected %d, got %d", testCase.want, got)
			}
		})
	}
}

func TestBuildLinuxFirstBootIdentityCommands(t *testing.T) {
	t.Parallel()

	commands := strings.Join(buildLinuxFirstBootIdentityCommands("vm-abcd1234"), "\n")
	requiredFragments := []string{
		"truncate -s 0 /etc/machine-id",
		"rm -f /var/lib/dbus/machine-id",
		"rm -f /var/lib/dhcp/*.leases",
		"rm -f /var/lib/NetworkManager/*.lease",
		"rm -f /var/lib/systemd/network/*.lease",
		"rm -rf /run/systemd/netif/leases/*",
		"rm -rf /var/lib/cloud/instances/* /var/lib/cloud/instance",
		"vm-abcd1234",
	}

	for _, fragment := range requiredFragments {
		if !strings.Contains(commands, fragment) {
			t.Fatalf("expected first boot identity commands to contain %q, got %s", fragment, commands)
		}
	}
}

func TestBuildFnOSIdentityCommands(t *testing.T) {
	t.Parallel()

	resetCommand := buildFnOSIdentityResetCommand()
	if !strings.Contains(resetCommand, "truncate -s 0 /etc/machine-id") {
		t.Fatalf("expected reset command to clear /etc/machine-id, got %s", resetCommand)
	}
	if !strings.Contains(resetCommand, "rm -f /var/lib/dbus/machine-id") {
		t.Fatalf("expected reset command to remove dbus machine-id, got %s", resetCommand)
	}

	preserveCommand := buildFnOSIdentityPreservationCommand()
	if strings.Contains(preserveCommand, "truncate -s 0 /etc/machine-id") {
		t.Fatalf("preserve command must not clear /etc/machine-id, got %s", preserveCommand)
	}
	if !strings.Contains(preserveCommand, "cp /etc/machine-id /var/lib/dbus/machine-id") {
		t.Fatalf("expected preserve command to sync dbus machine-id, got %s", preserveCommand)
	}

	customCommand, err := buildFnOSCustomDeviceIDCommand("679ca7cf8fe242c4a64141a25a68f677")
	if err != nil {
		t.Fatalf("expected custom device id command to be valid, got %v", err)
	}
	if !strings.Contains(customCommand, "CUSTOM_DEVICE_ID='679ca7cf8fe242c4a64141a25a68f677'") {
		t.Fatalf("expected custom command to set /etc/device_id value, got %s", customCommand)
	}
	if !strings.Contains(customCommand, "CUSTOM_MACHINE_ID='679ca7cf8fe242c4a64141a25a68f67700000000'") {
		t.Fatalf("expected 32-char custom id to be expanded to trim machine id, got %s", customCommand)
	}
	if !strings.Contains(customCommand, "> /etc/device_id") || !strings.Contains(customCommand, "> /usr/trim/etc/machine_id") {
		t.Fatalf("expected custom command to write FnOS id files, got %s", customCommand)
	}
	if err := ValidateFnOSDeviceID("not-a-device-id"); err == nil {
		t.Fatalf("expected invalid FnOS device ID to be rejected")
	}
}
