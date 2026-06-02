package service

import (
	"strings"
	"testing"
)

func TestNormalizeVMBootType(t *testing.T) {
	if got := NormalizeVMBootType(" UEFI-Secure "); got != VMBootTypeUEFISecure {
		t.Fatalf("NormalizeVMBootType returned %q, want %q", got, VMBootTypeUEFISecure)
	}
	if got := NormalizeVMBootType("unknown"); got != "" {
		t.Fatalf("NormalizeVMBootType returned %q, want empty", got)
	}
}

func TestParseVMBootTypeFromDomainXML(t *testing.T) {
	tests := []struct {
		name string
		xml  string
		want string
	}{
		{
			name: "bios",
			xml:  "<domain><os><type arch='x86_64'>hvm</type></os></domain>",
			want: VMBootTypeBIOS,
		},
		{
			name: "uefi",
			xml:  "<domain><os firmware='efi'><type arch='x86_64'>hvm</type></os></domain>",
			want: VMBootTypeUEFI,
		},
		{
			name: "uefi secure",
			xml: "<domain><os firmware='efi'><type arch='x86_64'>hvm</type><firmware><feature enabled='yes' name='secure-boot'/></firmware>" +
				"<loader readonly='yes' secure='yes' type='pflash'>/usr/share/OVMF/OVMF_CODE_4M.ms.fd</loader></os></domain>",
			want: VMBootTypeUEFISecure,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := ParseVMBootTypeFromDomainXML(tc.xml); got != tc.want {
				t.Fatalf("ParseVMBootTypeFromDomainXML() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestParseVMArchAndMachineTypeFromDomainXML(t *testing.T) {
	xml := "<domain><os><type arch='x86_64' machine='pc-q35-9.0'>hvm</type></os></domain>"
	if got := ParseVMArchFromDomainXML(xml); got != "x86_64" {
		t.Fatalf("ParseVMArchFromDomainXML() = %q, want %q", got, "x86_64")
	}
	if got := ParseVMMachineTypeFromDomainXML(xml); got != "q35" {
		t.Fatalf("ParseVMMachineTypeFromDomainXML() = %q, want %q", got, "q35")
	}
}

func TestApplyVMBootTypeToDomainXMLSwitchToUEFI(t *testing.T) {
	xml := `<domain type='kvm'>
  <name>demo</name>
  <os>
    <type arch='x86_64' machine='pc-q35-9.0'>hvm</type>
    <boot dev='hd'/>
  </os>
  <features>
    <acpi/>
    <apic/>
  </features>
  <devices/>
</domain>`

	updated, err := ApplyVMBootTypeToDomainXML("demo", xml, VMBootTypeUEFI)
	if err != nil {
		t.Fatalf("ApplyVMBootTypeToDomainXML returned error: %v", err)
	}
	if !strings.Contains(updated, "<os firmware='efi'>") {
		t.Fatalf("expected EFI firmware attribute, got: %s", updated)
	}
	if !strings.Contains(updated, "<loader readonly='yes' type='pflash'>") {
		t.Fatalf("expected non-secure loader, got: %s", updated)
	}
	if !strings.Contains(updated, "/var/lib/libvirt/qemu/nvram/demo_VARS.fd") {
		t.Fatalf("expected generated nvram path, got: %s", updated)
	}
	if !strings.Contains(updated, "format='qcow2'") {
		t.Fatalf("expected qcow2 nvram format, got: %s", updated)
	}
	if strings.Contains(updated, "secure-boot") {
		t.Fatalf("did not expect secure boot feature in plain UEFI XML: %s", updated)
	}
}

func TestApplyVMBootTypeToDomainXMLStripsSecureBootForPlainUEFI(t *testing.T) {
	xml := `<domain type='kvm'>
  <name>demo</name>
  <os firmware='efi'>
    <type arch='x86_64' machine='pc-q35-10.2'>hvm</type>
    <firmware>
      <feature enabled='yes' name='enrolled-keys'/>
      <feature enabled='yes' name='secure-boot'/>
    </firmware>
    <loader readonly='yes' secure='yes' type='pflash'>/usr/share/OVMF/OVMF_CODE_4M.ms.fd</loader>
    <nvram template='/usr/share/OVMF/OVMF_VARS_4M.ms.fd'>/var/lib/libvirt/qemu/nvram/demo_VARS.fd</nvram>
    <boot dev='hd'/>
  </os>
  <features>
    <acpi/>
    <apic/>
    <smm state='on'/>
  </features>
  <devices/>
</domain>`

	updated, err := ApplyVMBootTypeToDomainXML("demo", xml, VMBootTypeUEFI)
	if err != nil {
		t.Fatalf("ApplyVMBootTypeToDomainXML returned error: %v", err)
	}
	if strings.Contains(updated, "secure-boot") || strings.Contains(updated, "secure='yes'") {
		t.Fatalf("plain UEFI should strip secure boot settings, got: %s", updated)
	}
	if !strings.Contains(updated, "OVMF_CODE_4M.fd") {
		t.Fatalf("expected non-secure OVMF loader, got: %s", updated)
	}
	if got := ParseVMBootTypeFromDomainXML(updated); got != VMBootTypeUEFI {
		t.Fatalf("ParseVMBootTypeFromDomainXML() = %q, want %q", got, VMBootTypeUEFI)
	}
	if !strings.Contains(updated, "format='qcow2'") {
		t.Fatalf("expected plain UEFI normalization to use qcow2 nvram, got: %s", updated)
	}
}

func TestApplyVMBootTypeToDomainXMLSwitchToSecureBoot(t *testing.T) {
	xml := `<domain type='kvm'>
  <name>demo</name>
  <os firmware='efi'>
    <type arch='x86_64' machine='pc-q35-9.0'>hvm</type>
    <loader readonly='yes' type='pflash'>/usr/share/OVMF/OVMF_CODE_4M.fd</loader>
    <nvram template='/usr/share/OVMF/OVMF_VARS_4M.fd'>/var/lib/libvirt/qemu/nvram/demo_VARS.fd</nvram>
    <boot dev='hd'/>
  </os>
  <features>
    <acpi/>
  </features>
  <devices/>
</domain>`

	updated, err := ApplyVMBootTypeToDomainXML("demo", xml, VMBootTypeUEFISecure)
	if err != nil {
		t.Fatalf("ApplyVMBootTypeToDomainXML returned error: %v", err)
	}
	if !strings.Contains(updated, "name='secure-boot'") {
		t.Fatalf("expected secure boot feature, got: %s", updated)
	}
	if !strings.Contains(updated, "secure='yes'") {
		t.Fatalf("expected secure loader flag, got: %s", updated)
	}
	if !strings.Contains(updated, "<smm state='on'/>") {
		t.Fatalf("expected SMM enabled for secure boot, got: %s", updated)
	}
	if !strings.Contains(updated, "format='qcow2'") {
		t.Fatalf("expected secure UEFI normalization to use qcow2 nvram, got: %s", updated)
	}
}

func TestSetDomainNVRAMFormat(t *testing.T) {
	xml := `<domain><os firmware='efi'><nvram template='/tmp/base.fd' templateFormat='raw' format='raw'>/tmp/demo.fd</nvram></os></domain>`
	updated := setDomainNVRAMFormat(xml, "qcow2")
	if strings.Contains(updated, "format='raw'") || !strings.Contains(updated, "format='qcow2'") {
		t.Fatalf("setDomainNVRAMFormat should replace existing format, got: %s", updated)
	}

	withoutFormat := `<domain><os firmware='efi'><nvram template='/tmp/base.fd'>/tmp/demo.fd</nvram></os></domain>`
	updated = setDomainNVRAMFormat(withoutFormat, "qcow2")
	if !strings.Contains(updated, "<nvram format='qcow2' template='/tmp/base.fd'>") {
		t.Fatalf("setDomainNVRAMFormat should insert missing format, got: %s", updated)
	}
}

func TestApplyVMBootTypeToDomainXMLSwitchToBIOS(t *testing.T) {
	xml := `<domain type='kvm'>
  <name>demo</name>
  <os firmware='efi'>
    <type arch='x86_64' machine='pc-q35-9.0'>hvm</type>
    <firmware>
      <feature enabled='yes' name='secure-boot'/>
    </firmware>
    <loader readonly='yes' secure='yes' type='pflash'>/usr/share/OVMF/OVMF_CODE_4M.ms.fd</loader>
    <nvram template='/usr/share/OVMF/OVMF_VARS_4M.ms.fd'>/var/lib/libvirt/qemu/nvram/demo_VARS.fd</nvram>
    <boot dev='hd'/>
  </os>
  <features>
    <acpi/>
    <smm state='on'/>
  </features>
  <devices/>
</domain>`

	updated, err := ApplyVMBootTypeToDomainXML("demo", xml, VMBootTypeBIOS)
	if err != nil {
		t.Fatalf("ApplyVMBootTypeToDomainXML returned error: %v", err)
	}
	if strings.Contains(updated, "firmware='efi'") {
		t.Fatalf("expected firmware attribute removed, got: %s", updated)
	}
	if strings.Contains(updated, "<loader") || strings.Contains(updated, "<nvram") {
		t.Fatalf("expected UEFI loader and nvram removed, got: %s", updated)
	}
}

func TestApplyVMBootTypeToDomainXMLRejectsUnsupportedSecureBootMachine(t *testing.T) {
	xml := `<domain><os><type arch='x86_64' machine='pc-i440fx-9.0'>hvm</type></os></domain>`
	if _, err := ApplyVMBootTypeToDomainXML("demo", xml, VMBootTypeUEFISecure); err == nil {
		t.Fatalf("expected secure boot on i440fx to be rejected")
	}
}
