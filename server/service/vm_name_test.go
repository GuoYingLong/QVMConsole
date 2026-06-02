package service

import "testing"

func TestValidateVMName(t *testing.T) {
	validNames := []string{"vm01", "Test123", "abc"}
	for _, name := range validNames {
		if err := ValidateVMName(name); err != nil {
			t.Fatalf("expected %q to be valid, got error: %v", name, err)
		}
	}

	invalidNames := []string{"", "vm-01", "vm_01", "vm 01", "测试"}
	for _, name := range invalidNames {
		if err := ValidateVMName(name); err == nil {
			t.Fatalf("expected %q to be invalid", name)
		}
	}
}

func TestValidateVMNamePrefix(t *testing.T) {
	if err := ValidateVMNamePrefix("batch01"); err != nil {
		t.Fatalf("expected prefix to be valid, got error: %v", err)
	}
	if err := ValidateVMNamePrefix("batch-01"); err == nil {
		t.Fatalf("expected prefix with hyphen to be invalid")
	}
}

func TestGenerateRandomVMName(t *testing.T) {
	name := GenerateRandomVMName()
	if err := ValidateVMName(name); err != nil {
		t.Fatalf("generated vm name should be valid, got error: %v", err)
	}
	if len(name) != 10 {
		t.Fatalf("unexpected vm name length: %d", len(name))
	}
}
