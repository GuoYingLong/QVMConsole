package service

import (
	"strings"
	"testing"
)

func TestResolveVMAPICEnabled(t *testing.T) {
	if !ResolveVMAPICEnabled(nil) {
		t.Fatalf("expected nil APIC config to default to enabled")
	}

	disabled := false
	if ResolveVMAPICEnabled(&disabled) {
		t.Fatalf("expected explicit false APIC config to disable APIC")
	}
}

func TestParseVMAPICFromDomainXML(t *testing.T) {
	if !ParseVMAPICFromDomainXML("<domain><features><acpi/><apic/></features></domain>") {
		t.Fatalf("expected APIC to be detected")
	}
	if ParseVMAPICFromDomainXML("<domain><features><acpi/></features></domain>") {
		t.Fatalf("expected APIC to be absent")
	}
}

func TestApplyVMAPICToDomainXMLEnableExistingFeatures(t *testing.T) {
	xmlContent := "<domain>\n  <features>\n    <acpi/>\n  </features>\n  <devices/>\n</domain>"
	enabled := true

	updated, err := ApplyVMAPICToDomainXML(xmlContent, &enabled)
	if err != nil {
		t.Fatalf("ApplyVMAPICToDomainXML returned error: %v", err)
	}
	if !strings.Contains(updated, "<apic/>") {
		t.Fatalf("expected APIC node to be inserted, got: %s", updated)
	}
}

func TestApplyVMAPICToDomainXMLDisable(t *testing.T) {
	xmlContent := "<domain>\n  <features>\n    <acpi/>\n    <apic/>\n  </features>\n</domain>"
	disabled := false

	updated, err := ApplyVMAPICToDomainXML(xmlContent, &disabled)
	if err != nil {
		t.Fatalf("ApplyVMAPICToDomainXML returned error: %v", err)
	}
	if strings.Contains(updated, "<apic/>") {
		t.Fatalf("expected APIC node to be removed, got: %s", updated)
	}
}

func TestApplyVMAPICToDomainXMLInsertFeaturesBlock(t *testing.T) {
	xmlContent := "<domain>\n  <devices/>\n</domain>"
	enabled := true

	updated, err := ApplyVMAPICToDomainXML(xmlContent, &enabled)
	if err != nil {
		t.Fatalf("ApplyVMAPICToDomainXML returned error: %v", err)
	}
	if !strings.Contains(updated, "<features>") || !strings.Contains(updated, "<apic/>") {
		t.Fatalf("expected features block with APIC to be inserted, got: %s", updated)
	}
}
