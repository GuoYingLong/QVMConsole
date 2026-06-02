package service

import (
	"reflect"
	"testing"
)

func TestParseHostDiskDeviceNames(t *testing.T) {
	output := "sda disk\nsda1 part\nnvme0n1 disk\ndm-0 lvm\nvda disk\n"

	got := parseHostDiskDeviceNames(output)
	want := []string{"sda", "nvme0n1", "vda"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected devices %v, got %v", want, got)
	}
}

func TestDetectTopLevelDiskDevicesFromDiskstats(t *testing.T) {
	content := `8 0 sda 1 0 100 0 2 0 200 0 0 0 0 0 0 0 0 0 0
8 1 sda1 1 0 100 0 2 0 200 0 0 0 0 0 0 0 0 0 0
252 0 dm-0 1 0 100 0 2 0 200 0 0 0 0 0 0 0 0 0 0
259 0 nvme0n1 1 0 100 0 2 0 200 0 0 0 0 0 0 0 0 0 0
253 0 vda 1 0 100 0 2 0 200 0 0 0 0 0 0 0 0 0 0
202 0 xvda 1 0 100 0 2 0 200 0 0 0 0 0 0 0 0 0 0`

	got := detectTopLevelDiskDevicesFromDiskstats(content)
	want := []string{"sda", "nvme0n1", "vda", "xvda"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected fallback devices %v, got %v", want, got)
	}
}

func TestParseDiskStatsSectors(t *testing.T) {
	content := `8 0 sda 10 0 120 0 11 0 240 0 0 0 0 0 0 0 0 0 0
259 0 nvme0n1 12 0 300 0 13 0 500 0 0 0 0 0 0 0 0 0 0
8 1 sda1 99 0 999 0 99 0 999 0 0 0 0 0 0 0 0 0 0`

	readSectors, writeSectors := parseDiskStatsSectors(content, []string{"sda", "nvme0n1"})

	if readSectors != 420 {
		t.Fatalf("expected read sectors 420, got %d", readSectors)
	}
	if writeSectors != 740 {
		t.Fatalf("expected write sectors 740, got %d", writeSectors)
	}
}
