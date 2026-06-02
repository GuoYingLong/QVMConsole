package service

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"kvm_console/utils"
)

func resetTemplateDiskInfoCacheForTest() {
	templateDiskInfoCache.Lock()
	templateDiskInfoCache.items = make(map[string]templateDiskInfoCacheEntry)
	templateDiskInfoCache.Unlock()
}

func TestFillTemplateInfoSizesUsesCachedDiskInfo(t *testing.T) {
	tempDir := t.TempDir()
	templatePath := filepath.Join(tempDir, "demo.qcow2")
	if err := os.WriteFile(templatePath, []byte("demo"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	oldCommand := templateDiskInfoCommand
	oldStat := templateDiskInfoStat
	resetTemplateDiskInfoCacheForTest()
	defer func() {
		templateDiskInfoCommand = oldCommand
		templateDiskInfoStat = oldStat
		resetTemplateDiskInfoCacheForTest()
	}()

	callCount := 0
	templateDiskInfoStat = os.Stat
	templateDiskInfoCommand = func(path string) *utils.CmdResult {
		callCount++
		return &utils.CmdResult{Stdout: `{"actual-size":1073741824,"virtual-size":4294967296}`}
	}

	first := TemplateInfo{Path: templatePath}
	fillTemplateInfoSizes(&first)
	second := TemplateInfo{Path: templatePath}
	fillTemplateInfoSizes(&second)

	if callCount != 1 {
		t.Fatalf("templateDiskInfoCommand 调用次数 = %d，期望 1", callCount)
	}
	if first.ActualSize != "1.00 GB" || second.ActualSize != "1.00 GB" {
		t.Fatalf("ActualSize = %q / %q，期望都为 1.00 GB", first.ActualSize, second.ActualSize)
	}
	if first.VirtualSize != "4.00 GiB" || second.VirtualSize != "4.00 GiB" {
		t.Fatalf("VirtualSize = %q / %q，期望都为 4.00 GiB", first.VirtualSize, second.VirtualSize)
	}
}

func TestFillTemplateInfoSizesRefreshesCacheWhenFileChanges(t *testing.T) {
	tempDir := t.TempDir()
	templatePath := filepath.Join(tempDir, "demo.qcow2")
	if err := os.WriteFile(templatePath, []byte("demo"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	oldCommand := templateDiskInfoCommand
	oldStat := templateDiskInfoStat
	resetTemplateDiskInfoCacheForTest()
	defer func() {
		templateDiskInfoCommand = oldCommand
		templateDiskInfoStat = oldStat
		resetTemplateDiskInfoCacheForTest()
	}()

	callCount := 0
	templateDiskInfoStat = os.Stat
	templateDiskInfoCommand = func(path string) *utils.CmdResult {
		callCount++
		if callCount == 1 {
			return &utils.CmdResult{Stdout: `{"actual-size":1073741824,"virtual-size":4294967296}`}
		}
		return &utils.CmdResult{Stdout: `{"actual-size":2147483648,"virtual-size":8589934592}`}
	}

	first := TemplateInfo{Path: templatePath}
	fillTemplateInfoSizes(&first)

	if err := os.WriteFile(templatePath, []byte("demo-updated"), 0o644); err != nil {
		t.Fatalf("WriteFile() update error = %v", err)
	}
	newTime := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(templatePath, newTime, newTime); err != nil {
		t.Fatalf("Chtimes() error = %v", err)
	}

	second := TemplateInfo{Path: templatePath}
	fillTemplateInfoSizes(&second)

	if callCount != 2 {
		t.Fatalf("templateDiskInfoCommand 调用次数 = %d，期望 2", callCount)
	}
	if second.ActualSize != "2.00 GB" {
		t.Fatalf("更新后 ActualSize = %q，期望 2.00 GB", second.ActualSize)
	}
	if second.VirtualSize != "8.00 GiB" {
		t.Fatalf("更新后 VirtualSize = %q，期望 8.00 GiB", second.VirtualSize)
	}
}

func TestFillTemplateInfoSizesBatchPopulatesAllTemplates(t *testing.T) {
	tempDir := t.TempDir()
	firstPath := filepath.Join(tempDir, "first.qcow2")
	secondPath := filepath.Join(tempDir, "second.qcow2")
	if err := os.WriteFile(firstPath, []byte("first"), 0o644); err != nil {
		t.Fatalf("WriteFile(first) error = %v", err)
	}
	if err := os.WriteFile(secondPath, []byte("second"), 0o644); err != nil {
		t.Fatalf("WriteFile(second) error = %v", err)
	}

	oldCommand := templateDiskInfoCommand
	oldStat := templateDiskInfoStat
	resetTemplateDiskInfoCacheForTest()
	defer func() {
		templateDiskInfoCommand = oldCommand
		templateDiskInfoStat = oldStat
		resetTemplateDiskInfoCacheForTest()
	}()

	templateDiskInfoStat = os.Stat
	templateDiskInfoCommand = func(path string) *utils.CmdResult {
		switch filepath.Base(path) {
		case "first.qcow2":
			return &utils.CmdResult{Stdout: `{"actual-size":1073741824,"virtual-size":4294967296}`}
		case "second.qcow2":
			return &utils.CmdResult{Stdout: `{"actual-size":536870912,"virtual-size":2147483648}`}
		default:
			return &utils.CmdResult{Stdout: `{}`}
		}
	}

	templates := []TemplateInfo{
		{Path: firstPath},
		{Path: secondPath},
	}
	fillTemplateInfoSizesBatch(templates)

	if templates[0].VirtualSize != "4.00 GiB" {
		t.Fatalf("第一个模板 VirtualSize = %q，期望 4.00 GiB", templates[0].VirtualSize)
	}
	if templates[1].VirtualSize != "2.00 GiB" {
		t.Fatalf("第二个模板 VirtualSize = %q，期望 2.00 GiB", templates[1].VirtualSize)
	}
}
