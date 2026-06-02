package service

import (
	"strings"
	"testing"
)

func TestNormalizeSMBIOS1ConfigBase64(t *testing.T) {
	cfg, err := NormalizeSMBIOS1Config(&VMSMBIOS1Config{
		Base64:       true,
		Manufacturer: "T3BlbkFJ",
		Product:      "S1ZNQ29uc29sZQ==",
	})
	if err != nil {
		t.Fatalf("NormalizeSMBIOS1Config returned error: %v", err)
	}

	if cfg.Manufacturer != "OpenAI" {
		t.Fatalf("unexpected manufacturer: %s", cfg.Manufacturer)
	}
	if cfg.Product != "KVMConsole" {
		t.Fatalf("unexpected product: %s", cfg.Product)
	}
}

func TestApplySMBIOS1ConfigToDomainXML(t *testing.T) {
	xmlContent := `<domain type='kvm'>
  <name>test</name>
  <uuid>df356327-4f8f-409c-bfe9-98c91465ccf0</uuid>
  <memory unit='KiB'>2097152</memory>
  <vcpu placement='static'>2</vcpu>
  <os>
    <type arch='x86_64'>hvm</type>
  </os>
</domain>`

	cfg := &VMSMBIOS1Config{
		Manufacturer: "OpenAI",
		Product:      "KVMConsole",
		UUID:         "df356327-4f8f-409c-bfe9-98c91465ccf0",
	}

	updated, err := ApplySMBIOS1ConfigToDomainXML(xmlContent, cfg, false)
	if err != nil {
		t.Fatalf("ApplySMBIOS1ConfigToDomainXML returned error: %v", err)
	}

	expectedParts := []string{
		"<sysinfo type='smbios'>",
		"<entry name='manufacturer'>OpenAI</entry>",
		"<entry name='product'>KVMConsole</entry>",
		"<entry name='uuid'>df356327-4f8f-409c-bfe9-98c91465ccf0</entry>",
		"<smbios mode='sysinfo'/>",
	}
	for _, part := range expectedParts {
		if !strings.Contains(updated, part) {
			t.Fatalf("updated xml missing %q:\n%s", part, updated)
		}
	}
}

func TestApplySMBIOS1ConfigRejectUUIDMismatch(t *testing.T) {
	xmlContent := `<domain type='kvm'>
  <name>test</name>
  <uuid>df356327-4f8f-409c-bfe9-98c91465ccf0</uuid>
  <memory unit='KiB'>2097152</memory>
  <vcpu placement='static'>2</vcpu>
  <os>
    <type arch='x86_64'>hvm</type>
  </os>
</domain>`

	_, err := ApplySMBIOS1ConfigToDomainXML(xmlContent, &VMSMBIOS1Config{
		UUID: "11111111-2222-3333-4444-555555555555",
	}, false)
	if err == nil {
		t.Fatal("expected UUID mismatch error, got nil")
	}
}

func TestParseSMBIOS1ConfigFromDomainXML(t *testing.T) {
	xmlContent := `<domain type='kvm'>
  <name>test</name>
  <uuid>df356327-4f8f-409c-bfe9-98c91465ccf0</uuid>
  <sysinfo type='smbios'>
    <system>
      <entry name='manufacturer'>OpenAI</entry>
      <entry name='product'>KVMConsole</entry>
      <entry name='version'>v1</entry>
      <entry name='serial'>SN123</entry>
      <entry name='uuid'>df356327-4f8f-409c-bfe9-98c91465ccf0</entry>
      <entry name='sku'>SKU-1</entry>
      <entry name='family'>TestFamily</entry>
    </system>
  </sysinfo>
  <os>
    <type arch='x86_64'>hvm</type>
    <smbios mode='sysinfo'/>
  </os>
</domain>`

	cfg := ParseSMBIOS1ConfigFromDomainXML(xmlContent)
	if cfg == nil {
		t.Fatal("expected parsed config, got nil")
	}

	if cfg.Manufacturer != "OpenAI" || cfg.Product != "KVMConsole" || cfg.UUID != "df356327-4f8f-409c-bfe9-98c91465ccf0" {
		t.Fatalf("unexpected parsed config: %+v", cfg)
	}
}
