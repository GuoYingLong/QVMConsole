package service

import (
	"reflect"
	"testing"
)

func TestParseGuestDiskLayoutWithWindowsRecoveryPartition(t *testing.T) {
	output := `__PARTTYPE__
gpt
__SECTOR_SIZE__
512
__DISK_SECTORS__
41943040
__PART_LIST__
[0] = {
  part_num: 1
  part_start: 1048576
  part_end: 105906175
  part_size: 104857600
}
[1] = {
  part_num: 2
  part_start: 105906176
  part_end: 122683391
  part_size: 16777216
}
[2] = {
  part_num: 3
  part_start: 122683392
  part_end: 20647510015
  part_size: 20524826624
}
[3] = {
  part_num: 4
  part_start: 20647510016
  part_end: 21472739327
  part_size: 825229312
}
__FILESYSTEMS__
/dev/sda1: vfat
/dev/sda3: ntfs
/dev/sda4: ntfs
`

	layout, err := parseGuestDiskLayout(output, "/dev/sda")
	if err != nil {
		t.Fatalf("parseGuestDiskLayout() error = %v", err)
	}
	if layout.PartType != "gpt" {
		t.Fatalf("PartType = %q, want gpt", layout.PartType)
	}
	if layout.SectorSize != 512 {
		t.Fatalf("SectorSize = %d, want 512", layout.SectorSize)
	}
	if len(layout.Partitions) != 4 {
		t.Fatalf("partition count = %d, want 4", len(layout.Partitions))
	}
	if layout.Partitions[2].FileSystem != "ntfs" {
		t.Fatalf("partition 3 filesystem = %q, want ntfs", layout.Partitions[2].FileSystem)
	}

	metaOutput := `__PART_META_1__
C12A7328-F81F-11D2-BA4B-00A0C93EC93B
11111111-1111-1111-1111-111111111111
EFI system partition
0
__PART_META_2__
E3C9E316-0B5C-4DB8-817D-F92DF00215AE
22222222-2222-2222-2222-222222222222
Microsoft reserved partition
0
__PART_META_3__
EBD0A0A2-B9E5-4433-87C0-68B6B72699C7
33333333-3333-3333-3333-333333333333
Basic data partition
0
__PART_META_4__
DE94BBA4-06D1-4D40-A16A-BFD50179D6AC
44444444-4444-4444-4444-444444444444

1
`
	if err := applyGuestDiskPartitionMeta(layout, metaOutput); err != nil {
		t.Fatalf("applyGuestDiskPartitionMeta() error = %v", err)
	}

	osPart := layout.findWindowsOSPartition()
	if osPart == nil || osPart.Num != 3 {
		t.Fatalf("findWindowsOSPartition() = %+v, want partition 3", osPart)
	}
	recoveryPart := layout.findRecoveryPartitionAfter(osPart)
	if recoveryPart == nil || recoveryPart.Num != 4 {
		t.Fatalf("findRecoveryPartitionAfter() = %+v, want partition 4", recoveryPart)
	}
}

func TestParseGuestDiskLayoutWithMBRPartitionTable(t *testing.T) {
	output := `__PARTTYPE__
msdos
__SECTOR_SIZE__
512
__DISK_SECTORS__
31457280
__PART_LIST__
[0] = {
  part_num: 1
  part_start: 1048576
  part_end: 16106127359
  part_size: 16000000000
}
__FILESYSTEMS__
/dev/sda1: ntfs
`

	layout, err := parseGuestDiskLayout(output, "/dev/sda")
	if err != nil {
		t.Fatalf("parseGuestDiskLayout() error = %v", err)
	}
	if layout.PartType != "msdos" {
		t.Fatalf("PartType = %q, want msdos", layout.PartType)
	}
	if len(layout.Partitions) != 1 {
		t.Fatalf("partition count = %d, want 1", len(layout.Partitions))
	}
	if layout.Partitions[0].FileSystem != "ntfs" {
		t.Fatalf("partition 1 filesystem = %q, want ntfs", layout.Partitions[0].FileSystem)
	}
	if isWindowsRecoveryPartition(&layout.Partitions[0]) {
		t.Fatalf("MBR NTFS system partition should not be treated as recovery partition")
	}
	if got := layout.lastUsableSector(); got != 31457279 {
		t.Fatalf("lastUsableSector() = %d, want 31457279", got)
	}
}

func TestBuildExpandLastWindowsPartitionCommands(t *testing.T) {
	gptLayout := &guestDiskLayout{Device: "/dev/sda", PartType: "gpt"}
	mbrLayout := &guestDiskLayout{Device: "/dev/sda", PartType: "msdos"}
	osPart := guestDiskPartition{Num: 2}

	gptCommands := buildExpandLastWindowsPartitionCommands(gptLayout, osPart, 41943006)
	wantGPT := []string{
		"run",
		"part-expand-gpt /dev/sda",
		"part-resize /dev/sda 2 41943006",
		"blockdev-rereadpt /dev/sda",
	}
	if !reflect.DeepEqual(gptCommands, wantGPT) {
		t.Fatalf("gpt commands = %#v, want %#v", gptCommands, wantGPT)
	}

	mbrCommands := buildExpandLastWindowsPartitionCommands(mbrLayout, osPart, 41943039)
	wantMBR := []string{
		"run",
		"part-resize /dev/sda 2 41943039",
		"blockdev-rereadpt /dev/sda",
	}
	if !reflect.DeepEqual(mbrCommands, wantMBR) {
		t.Fatalf("mbr commands = %#v, want %#v", mbrCommands, wantMBR)
	}
}
