package service

import (
	"reflect"
	"testing"
)

func TestFindFnOSSystemPartitionSelectsLargestExtPartition(t *testing.T) {
	layout := &guestDiskLayout{
		Partitions: []guestDiskPartition{
			{Num: 1, FileSystem: "vfat", SizeBytes: 96 * 1024 * 1024},
			{Num: 2, FileSystem: "ext4", SizeBytes: 10 * 1024 * 1024 * 1024},
			{Num: 3, FileSystem: "ext4", SizeBytes: 1024 * 1024},
		},
	}

	part := layout.findFnOSSystemPartition()
	if part == nil || part.Num != 2 {
		t.Fatalf("findFnOSSystemPartition() = %+v, want partition 2", part)
	}
}

func TestBuildFnOSSystemDiskExpansionCommands(t *testing.T) {
	gptLayout := &guestDiskLayout{Device: "/dev/sda", PartType: "gpt"}
	mbrLayout := &guestDiskLayout{Device: "/dev/sda", PartType: "msdos"}
	systemPart := guestDiskPartition{Num: 2}

	gptCommands := buildFnOSSystemDiskExpansionCommands(gptLayout, systemPart, 25165790)
	wantGPT := []string{
		"run",
		"part-expand-gpt /dev/sda",
		"part-resize /dev/sda 2 25165790",
		"blockdev-rereadpt /dev/sda",
		"e2fsck-f /dev/sda2",
		"resize2fs /dev/sda2",
	}
	if !reflect.DeepEqual(gptCommands, wantGPT) {
		t.Fatalf("gpt commands = %#v, want %#v", gptCommands, wantGPT)
	}

	mbrCommands := buildFnOSSystemDiskExpansionCommands(mbrLayout, systemPart, 25165823)
	wantMBR := []string{
		"run",
		"part-resize /dev/sda 2 25165823",
		"blockdev-rereadpt /dev/sda",
		"e2fsck-f /dev/sda2",
		"resize2fs /dev/sda2",
	}
	if !reflect.DeepEqual(mbrCommands, wantMBR) {
		t.Fatalf("mbr commands = %#v, want %#v", mbrCommands, wantMBR)
	}
}
