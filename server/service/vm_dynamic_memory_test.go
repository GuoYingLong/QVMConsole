package service

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
)

func TestParseDomainMemoryXML(t *testing.T) {
	xmlStr := `<domain>
  <memory unit='KiB'>4194304</memory>
  <currentMemory unit='KiB'>2097152</currentMemory>
</domain>`
	values := parseDomainMemoryXML(xmlStr)
	if values.MemoryMB != 4096 {
		t.Fatalf("expected max memory 4096MB, got %d", values.MemoryMB)
	}
	if values.CurrentMemoryMB != 2048 {
		t.Fatalf("expected current memory 2048MB, got %d", values.CurrentMemoryMB)
	}
}

func TestApplyDynamicMemoryConfigToDomainXML(t *testing.T) {
	xmlStr := `<domain>
  <name>test</name>
  <memory unit='KiB'>2097152</memory>
  <vcpu>2</vcpu>
  <devices>
    <disk type='file' device='disk'></disk>
  </devices>
</domain>`

	got, err := ApplyDynamicMemoryConfigToDomainXML(xmlStr, 2048, 4096, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	values := parseDomainMemoryXML(got)
	if values.MemoryMB != 4096 {
		t.Fatalf("expected max memory 4096MB, got %d", values.MemoryMB)
	}
	if values.CurrentMemoryMB != 2048 {
		t.Fatalf("expected current memory 2048MB, got %d", values.CurrentMemoryMB)
	}
	if !hasUsableMemballoon(got) {
		t.Fatalf("expected usable memballoon in XML")
	}
}

func TestParseVMMemoryMetadataOutputEmpty(t *testing.T) {
	meta, err := parseVMMemoryMetadataOutput("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta != nil {
		t.Fatalf("expected empty metadata output to be treated as missing metadata")
	}
}

func TestParseVMMemoryMetadataOutputValid(t *testing.T) {
	want, err := newVMMemoryMetadata(2048, 1024, 4096, true, true)
	if err != nil {
		t.Fatalf("unexpected metadata build error: %v", err)
	}
	raw, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("unexpected json marshal error: %v", err)
	}
	output := `<memoryConfig xmlns:kvm-console-memory="https://kvm-console.local/domain-memory">` + base64.StdEncoding.EncodeToString(raw) + `</memoryConfig>`

	got, err := parseVMMemoryMetadataOutput(output)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil || !got.DynamicEnabled || !got.PendingApply || got.MemoryMaxMB != 4096 {
		t.Fatalf("unexpected metadata parse result: %#v", got)
	}
}

func TestApplyStaticMemoryConfigToDomainXML(t *testing.T) {
	xmlStr := `<domain>
  <name>test</name>
  <memory unit='KiB'>6291456</memory>
  <currentMemory unit='KiB'>4194304</currentMemory>
  <vcpu>2</vcpu>
  <devices>
    <memballoon model="virtio"><stats period="5"/></memballoon>
  </devices>
</domain>`

	got, err := ApplyStaticMemoryConfigToDomainXML(xmlStr, 4096)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	values := parseDomainMemoryXML(got)
	if values.MemoryMB != 4096 {
		t.Fatalf("expected static max memory 4096MB, got %d", values.MemoryMB)
	}
	if values.CurrentMemoryMB != 4096 {
		t.Fatalf("expected static current memory 4096MB, got %d", values.CurrentMemoryMB)
	}
}

func TestApplyVirtioMemConfigToDomainXML(t *testing.T) {
	xmlStr := `<domain>
  <name>test</name>
  <memory unit='KiB'>4194304</memory>
  <vcpu>4</vcpu>
  <cpu mode='host-passthrough' check='none' migratable='on'/>
  <devices>
    <disk type='file' device='disk'></disk>
    <memballoon model='virtio' freePageReporting='on'><stats period='5'/></memballoon>
  </devices>
</domain>`

	got, err := ApplyVirtioMemConfigToDomainXML(xmlStr, 2048, 4096)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	values := parseDomainMemoryXML(got)
	if values.MemoryMB != 2048 {
		t.Fatalf("expected base memory 2048MB, got %d", values.MemoryMB)
	}
	if values.CurrentMemoryMB != 2048 {
		t.Fatalf("expected current memory 2048MB, got %d", values.CurrentMemoryMB)
	}
	if !containsAll(got, "<maxMemory slots='16' unit='KiB'>4194304</maxMemory>", "<memory model='virtio-mem'>", "<requested unit='KiB'>0</requested>", "<numa>") {
		t.Fatalf("expected virtio-mem max memory, device and numa in XML:\n%s", got)
	}
	if containsAll(got, "freePageReporting='on'") {
		t.Fatalf("expected Windows virtio-mem XML to disable freePageReporting")
	}
}

func TestParseVirtioMemRequestedMB(t *testing.T) {
	xmlStr := `<domain>
  <devices>
    <memory model='virtio-mem'>
      <target>
        <size unit='KiB'>2097152</size>
        <node>0</node>
        <block unit='KiB'>2048</block>
        <requested unit='KiB'>32768</requested>
        <current unit='KiB'>32768</current>
      </target>
    </memory>
  </devices>
</domain>`

	if got := parseVirtioMemRequestedMB(xmlStr); got != 32 {
		t.Fatalf("expected requested 32MB, got %dMB", got)
	}
	if got := parseVirtioMemCurrentMB(xmlStr); got != 32 {
		t.Fatalf("expected current 32MB, got %dMB", got)
	}
}

func containsAll(value string, needles ...string) bool {
	for _, needle := range needles {
		if !strings.Contains(value, needle) {
			return false
		}
	}
	return true
}

func TestDynamicMemoryValidation(t *testing.T) {
	if _, err := newVMMemoryMetadata(4096, 8192, 4096, true, false); err == nil {
		t.Fatalf("expected min > initial validation error")
	}
	if _, err := newVMMemoryMetadata(4096, 1024, 2048, true, false); err == nil {
		t.Fatalf("expected initial > max validation error")
	}
}

func TestDefaultDynamicMemoryMaxGB(t *testing.T) {
	cases := map[int]int{
		1: 2,
		2: 3,
		4: 6,
		8: 11,
	}
	for initial, expected := range cases {
		if got := defaultDynamicMemoryMaxGB(initial); got != expected {
			t.Fatalf("expected initial %dGB max %dGB, got %dGB", initial, expected, got)
		}
	}
}

func TestCalculateVirtioMemScheduleTarget(t *testing.T) {
	cases := []struct {
		name      string
		actualMB  int
		usedMB    int
		initialMB int
		maxMB     int
		expected  int
	}{
		{
			name:      "usage above 70 percent expands by one GB",
			actualMB:  2048,
			usedMB:    1844,
			initialMB: 2048,
			maxMB:     6144,
			expected:  3072,
		},
		{
			name:      "expansion is capped by max memory",
			actualMB:  5632,
			usedMB:    4506,
			initialMB: 2048,
			maxMB:     6144,
			expected:  6144,
		},
		{
			name:      "usage below 50 percent shrinks to keep usage under 70 percent",
			actualMB:  6144,
			usedMB:    2458,
			initialMB: 2048,
			maxMB:     6144,
			expected:  3512,
		},
		{
			name:      "shrink never goes below base memory",
			actualMB:  4096,
			usedMB:    512,
			initialMB: 2048,
			maxMB:     6144,
			expected:  2048,
		},
		{
			name:      "usage between thresholds keeps current memory",
			actualMB:  4096,
			usedMB:    2458,
			initialMB: 2048,
			maxMB:     6144,
			expected:  4096,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := calculateVirtioMemScheduleTarget(tc.actualMB, tc.usedMB, tc.initialMB, tc.maxMB)
			if got != tc.expected {
				t.Fatalf("expected target %dMB, got %dMB", tc.expected, got)
			}
			if got < tc.actualMB && float64(tc.usedMB)/float64(got) > 0.70 {
				t.Fatalf("shrink target %dMB would leave usage above 70%%", got)
			}
		})
	}
}
