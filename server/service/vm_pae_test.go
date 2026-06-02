package service

import (
	"strings"
	"testing"
)

func TestResolveVMPAEEnabled(t *testing.T) {
	if !ResolveVMPAEEnabled(nil) {
		t.Fatalf("expected nil PAE config to default to enabled")
	}

	disabled := false
	if ResolveVMPAEEnabled(&disabled) {
		t.Fatalf("expected explicit false PAE config to disable PAE")
	}
}

func TestParseVMPAEFromDomainXML(t *testing.T) {
	if !ParseVMPAEFromDomainXML("<domain><features><acpi/><pae/></features></domain>") {
		t.Fatalf("expected PAE to be detected")
	}
	if ParseVMPAEFromDomainXML("<domain><features><acpi/></features></domain>") {
		t.Fatalf("expected PAE to be absent")
	}
}

func TestApplyVMPAEToDomainXMLEnableExistingFeatures(t *testing.T) {
	xmlContent := "<domain>\n  <os><type arch='x86_64'>hvm</type></os>\n  <features>\n    <acpi/>\n  </features>\n  <devices/>\n</domain>"
	enabled := true

	updated, err := ApplyVMPAEToDomainXML(xmlContent, &enabled)
	if err != nil {
		t.Fatalf("ApplyVMPAEToDomainXML returned error: %v", err)
	}
	if !strings.Contains(updated, "<pae/>") {
		t.Fatalf("expected PAE node to be inserted, got: %s", updated)
	}
}

func TestApplyVMPAEToDomainXMLDisable(t *testing.T) {
	xmlContent := "<domain>\n  <os><type arch='x86_64'>hvm</type></os>\n  <features>\n    <acpi/>\n    <pae/>\n  </features>\n</domain>"
	disabled := false

	updated, err := ApplyVMPAEToDomainXML(xmlContent, &disabled)
	if err != nil {
		t.Fatalf("ApplyVMPAEToDomainXML returned error: %v", err)
	}
	if strings.Contains(updated, "<pae/>") {
		t.Fatalf("expected PAE node to be removed, got: %s", updated)
	}
}

func TestApplyVMPAEToDomainXMLInsertFeaturesBlock(t *testing.T) {
	xmlContent := "<domain>\n  <os><type arch='x86_64'>hvm</type></os>\n  <devices/>\n</domain>"
	enabled := true

	updated, err := ApplyVMPAEToDomainXML(xmlContent, &enabled)
	if err != nil {
		t.Fatalf("ApplyVMPAEToDomainXML returned error: %v", err)
	}
	if !strings.Contains(updated, "<features>") || !strings.Contains(updated, "<pae/>") {
		t.Fatalf("expected features block with PAE to be inserted, got: %s", updated)
	}
}

func TestApplyVMPAEToDomainXMLIgnoreUnsupportedArch(t *testing.T) {
	xmlContent := "<domain>\n  <os><type arch='aarch64'>hvm</type></os>\n  <features>\n    <acpi/>\n  </features>\n</domain>"
	enabled := true

	updated, err := ApplyVMPAEToDomainXML(xmlContent, &enabled)
	if err != nil {
		t.Fatalf("ApplyVMPAEToDomainXML returned error: %v", err)
	}
	if strings.Contains(updated, "<pae/>") {
		t.Fatalf("expected unsupported arch to ignore PAE, got: %s", updated)
	}
}
