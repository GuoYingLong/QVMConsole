package service

import (
	"os"
	"path/filepath"
	"testing"

	"kvm_console/config"
)

func TestNormalizeVMDiskBus(t *testing.T) {
	cases := map[string]string{
		"":       "",
		"virtio": "virtio",
		"SCSI":   "scsi",
		"sata":   "sata",
		"IDE":    "ide",
		"foo":    "virtio",
	}
	for input, want := range cases {
		if got := NormalizeVMDiskBus(input); got != want {
			t.Fatalf("NormalizeVMDiskBus(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestNormalizeLoadedTemplateMetaNormalizesDefaultConfig(t *testing.T) {
	meta := normalizeLoadedTemplateMetaWithDetector("ubuntu-template", "/tmp/ubuntu-template.qcow2", &TemplateMeta{
		Type: "linux",
		DefaultConfig: &TemplateDefaultConfig{
			VCPU:                4,
			RAM:                 8,
			DiskSize:            64,
			DiskBus:             "SATA",
			NicModel:            "RTL8139",
			VideoModel:          "VMVGA",
			CPUTopologyMode:     "single_socket",
			FirstBootRebootMode: "cold",
		},
	}, true, func(string) string {
		return "bios"
	})

	if meta.DefaultConfig == nil {
		t.Fatalf("DefaultConfig = nil, want non-nil")
	}
	if meta.DefaultConfig.VCPU != 4 {
		t.Fatalf("DefaultConfig.VCPU = %d, want 4", meta.DefaultConfig.VCPU)
	}
	if meta.DefaultConfig.RAM != 8 {
		t.Fatalf("DefaultConfig.RAM = %d, want 8", meta.DefaultConfig.RAM)
	}
	if meta.DefaultConfig.DiskSize != 64 {
		t.Fatalf("DefaultConfig.DiskSize = %d, want 64", meta.DefaultConfig.DiskSize)
	}
	if meta.DefaultConfig.DiskBus != "sata" {
		t.Fatalf("DefaultConfig.DiskBus = %q, want %q", meta.DefaultConfig.DiskBus, "sata")
	}
	if meta.DefaultConfig.NicModel != "rtl8139" {
		t.Fatalf("DefaultConfig.NicModel = %q, want %q", meta.DefaultConfig.NicModel, "rtl8139")
	}
	if meta.DefaultConfig.VideoModel != "vmvga" {
		t.Fatalf("DefaultConfig.VideoModel = %q, want %q", meta.DefaultConfig.VideoModel, "vmvga")
	}
	if meta.DefaultConfig.CPUTopologyMode != "single_socket" {
		t.Fatalf("DefaultConfig.CPUTopologyMode = %q, want %q", meta.DefaultConfig.CPUTopologyMode, "single_socket")
	}
	if meta.DefaultConfig.FirstBootRebootMode != "cold" {
		t.Fatalf("DefaultConfig.FirstBootRebootMode = %q, want %q", meta.DefaultConfig.FirstBootRebootMode, "cold")
	}
}

func TestUpdateTemplatePublishCreatesMetaForLegacyTemplate(t *testing.T) {
	tempDir := t.TempDir()
	templateName := "legacy-template"
	templatePath := filepath.Join(tempDir, templateName+".qcow2")
	if err := os.WriteFile(templatePath, []byte("legacy"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	previousConfig := config.GlobalConfig
	config.GlobalConfig = &config.Config{TemplateDir: tempDir}
	defer func() {
		config.GlobalConfig = previousConfig
	}()

	err := UpdateTemplatePublish(templateName, &UpdateTemplatePublishParams{
		AdminName:           "Legacy Template",
		DisplayName:         "Legacy Display",
		CloneVisible:        true,
		Disabled:            false,
		Category:            "Debian",
		VCPU:                6,
		RAM:                 12,
		DiskSize:            120,
		DiskBus:             "SATA",
		NicModel:            "e1000e",
		VideoModel:          "VMVGA",
		CPUTopologyMode:     "single_socket",
		FirstBootRebootMode: "cold",
	})
	if err != nil {
		t.Fatalf("UpdateTemplatePublish() error = %v", err)
	}

	meta := loadTemplateMeta(templatePath)
	if meta == nil {
		t.Fatalf("loadTemplateMeta() = nil, want non-nil")
	}
	if meta.AdminName != "Legacy Template" {
		t.Fatalf("AdminName = %q, want %q", meta.AdminName, "Legacy Template")
	}
	if meta.DisplayName != "Legacy Display" {
		t.Fatalf("DisplayName = %q, want %q", meta.DisplayName, "Legacy Display")
	}
	if meta.Category != "Debian" {
		t.Fatalf("Category = %q, want %q", meta.Category, "Debian")
	}
	if !meta.CloneVisible {
		t.Fatalf("CloneVisible = false, want true")
	}
	if meta.DefaultConfig == nil {
		t.Fatalf("DefaultConfig = nil, want non-nil")
	}
	if meta.DefaultConfig.VCPU != 6 || meta.DefaultConfig.RAM != 12 || meta.DefaultConfig.DiskSize != 120 {
		t.Fatalf("DefaultConfig = %+v, want vcpu=6 ram=12 disk_size=120", *meta.DefaultConfig)
	}
	if meta.DefaultConfig.DiskBus != "sata" {
		t.Fatalf("DefaultConfig.DiskBus = %q, want %q", meta.DefaultConfig.DiskBus, "sata")
	}
	if meta.DefaultConfig.NicModel != "e1000e" {
		t.Fatalf("DefaultConfig.NicModel = %q, want %q", meta.DefaultConfig.NicModel, "e1000e")
	}
	if meta.DefaultConfig.VideoModel != "vmvga" {
		t.Fatalf("DefaultConfig.VideoModel = %q, want %q", meta.DefaultConfig.VideoModel, "vmvga")
	}
	if meta.DefaultConfig.CPUTopologyMode != "single_socket" {
		t.Fatalf("DefaultConfig.CPUTopologyMode = %q, want %q", meta.DefaultConfig.CPUTopologyMode, "single_socket")
	}
	if meta.DefaultConfig.FirstBootRebootMode != "cold" {
		t.Fatalf("DefaultConfig.FirstBootRebootMode = %q, want %q", meta.DefaultConfig.FirstBootRebootMode, "cold")
	}
}

func TestUpdateTemplateMetaPreservesDefaultConfigFields(t *testing.T) {
	tempDir := t.TempDir()
	templateName := "meta-template"
	templatePath := filepath.Join(tempDir, templateName+".qcow2")
	if err := os.WriteFile(templatePath, []byte("meta"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	previousConfig := config.GlobalConfig
	config.GlobalConfig = &config.Config{TemplateDir: tempDir}
	defer func() {
		config.GlobalConfig = previousConfig
	}()

	if err := UpdateTemplateMeta(templateName, &UpdateTemplateMetaParams{
		AdminName:           "Meta Template",
		DisplayName:         "Meta Display",
		CloneVisible:        true,
		Disabled:            false,
		Category:            "Debian",
		VCPU:                8,
		RAM:                 16,
		DiskSize:            200,
		DiskBus:             "SCSI",
		NicModel:            "RTL8139",
		VideoModel:          "Cirrus",
		CPUTopologyMode:     "host_default",
		FirstBootRebootMode: "cold",
	}); err != nil {
		t.Fatalf("UpdateTemplateMeta() error = %v", err)
	}

	meta := loadTemplateMeta(templatePath)
	if meta == nil {
		t.Fatalf("loadTemplateMeta() = nil, want non-nil")
	}
	if meta.DefaultConfig == nil {
		t.Fatalf("DefaultConfig = nil, want non-nil")
	}
	if meta.Category != "Debian" {
		t.Fatalf("Category = %q, want %q", meta.Category, "Debian")
	}
	if meta.DefaultConfig.VCPU != 8 || meta.DefaultConfig.RAM != 16 || meta.DefaultConfig.DiskSize != 200 {
		t.Fatalf("DefaultConfig = %+v, want vcpu=8 ram=16 disk_size=200", *meta.DefaultConfig)
	}
	if meta.DefaultConfig.DiskBus != "scsi" {
		t.Fatalf("DefaultConfig.DiskBus = %q, want %q", meta.DefaultConfig.DiskBus, "scsi")
	}
	if meta.DefaultConfig.NicModel != "rtl8139" {
		t.Fatalf("DefaultConfig.NicModel = %q, want %q", meta.DefaultConfig.NicModel, "rtl8139")
	}
	if meta.DefaultConfig.VideoModel != "cirrus" {
		t.Fatalf("DefaultConfig.VideoModel = %q, want %q", meta.DefaultConfig.VideoModel, "cirrus")
	}
	if meta.DefaultConfig.CPUTopologyMode != "host_default" {
		t.Fatalf("DefaultConfig.CPUTopologyMode = %q, want %q", meta.DefaultConfig.CPUTopologyMode, "host_default")
	}
	if meta.DefaultConfig.FirstBootRebootMode != "cold" {
		t.Fatalf("DefaultConfig.FirstBootRebootMode = %q, want %q", meta.DefaultConfig.FirstBootRebootMode, "cold")
	}
}
