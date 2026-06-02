package service

import "testing"

func TestValidateVMInactiveDomainXML(t *testing.T) {
	t.Run("accept valid domain xml", func(t *testing.T) {
		xmlContent := "<domain type='kvm'><name>demo</name><memory unit='MiB'>1024</memory></domain>"
		if err := ValidateVMInactiveDomainXML("demo", xmlContent); err != nil {
			t.Fatalf("ValidateVMInactiveDomainXML() error = %v", err)
		}
	})

	t.Run("reject empty xml", func(t *testing.T) {
		if err := ValidateVMInactiveDomainXML("demo", "   "); err == nil {
			t.Fatalf("expected empty xml to be rejected")
		}
	})

	t.Run("reject malformed xml", func(t *testing.T) {
		xmlContent := "<domain><name>demo</name>"
		if err := ValidateVMInactiveDomainXML("demo", xmlContent); err == nil {
			t.Fatalf("expected malformed xml to be rejected")
		}
	})

	t.Run("reject name mismatch", func(t *testing.T) {
		xmlContent := "<domain><name>other</name></domain>"
		if err := ValidateVMInactiveDomainXML("demo", xmlContent); err == nil {
			t.Fatalf("expected name mismatch to be rejected")
		}
	})
}
