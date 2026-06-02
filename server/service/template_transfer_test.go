package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"kvm_console/config"
)

func TestValidateTemplateName(t *testing.T) {
	t.Parallel()

	validNames := []string{"ubuntu-2404", "windows_2025", "template01", "ubuntu.24.04"}
	for _, name := range validNames {
		if err := ValidateTemplateName(name); err != nil {
			t.Fatalf("expected valid template name %q, got error: %v", name, err)
		}
	}

	invalidNames := []string{"", "bad name", "../escape", "template..cn"}
	for _, name := range invalidNames {
		if err := ValidateTemplateName(name); err == nil {
			t.Fatalf("expected invalid template name %q to fail", name)
		}
	}
}

func TestDetectTemplateTypeFromName(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"ubuntu-template": "linux",
		"win-server-2025": "windows",
		"NAS-fnos-demo":   "fnos",
	}
	for name, expected := range cases {
		if actual := detectTemplateTypeFromName(name); actual != expected {
			t.Fatalf("detectTemplateTypeFromName(%q) = %q, want %q", name, actual, expected)
		}
	}
}

func TestNormalizeTemplateCategory(t *testing.T) {
	t.Parallel()

	if actual := normalizeTemplateCategory("linux", ""); actual != "Ubuntu" {
		t.Fatalf("normalizeTemplateCategory linux default = %q, want %q", actual, "Ubuntu")
	}
	if actual := normalizeTemplateCategory("linux", "Debian"); actual != "Debian" {
		t.Fatalf("normalizeTemplateCategory linux explicit = %q, want %q", actual, "Debian")
	}
	if actual := normalizeTemplateCategory("linux", "Rocky Linux"); actual != "Ubuntu" {
		t.Fatalf("normalizeTemplateCategory unsupported linux category = %q, want %q", actual, "Ubuntu")
	}
	if actual := normalizeTemplateCategory("windows", ""); actual != "WindowsServer2022" {
		t.Fatalf("normalizeTemplateCategory windows default = %q, want %q", actual, "WindowsServer2022")
	}
	if actual := normalizeTemplateCategory("windows", "Windows10"); actual != "Windows10" {
		t.Fatalf("normalizeTemplateCategory windows explicit = %q, want %q", actual, "Windows10")
	}
	if actual := normalizeTemplateCategoryForName("windows", "", "WindowsServer2012R2"); actual != "WindowsServer2012R2" {
		t.Fatalf("normalizeTemplateCategoryForName windows 2012 r2 = %q, want %q", actual, "WindowsServer2012R2")
	}
	if actual := normalizeTemplateCategory("fnos", "Ubuntu"); actual != "" {
		t.Fatalf("normalizeTemplateCategory unsupported type = %q, want empty", actual)
	}
}

func TestValidateTemplateCategory(t *testing.T) {
	t.Parallel()

	if err := ValidateTemplateCategory("linux", "Debian"); err != nil {
		t.Fatalf("expected Debian category to be valid, got %v", err)
	}
	if err := ValidateTemplateCategory("linux", "Ubuntu 24.04"); err == nil {
		t.Fatalf("expected unsupported category to be rejected")
	}
	if err := ValidateTemplateCategory("windows", "WindowsServer2012R2"); err != nil {
		t.Fatalf("expected WindowsServer2012R2 category to be valid, got %v", err)
	}
	if err := ValidateTemplateCategory("windows", "Ubuntu"); err == nil {
		t.Fatalf("expected unsupported windows category to be rejected")
	}
	if err := ValidateTemplateCategory("fnos", "Ubuntu"); err == nil {
		t.Fatalf("expected unsupported type category to be rejected")
	}
}

func TestResolveTemplateBootType(t *testing.T) {
	t.Parallel()

	if actual, verified := resolveTemplateBootType("/tmp/demo.qcow2", "linux", "uefi", false, func(string) string {
		t.Fatal("expected detector to be skipped when boot type already exists")
		return ""
	}); actual != "uefi" || !verified {
		t.Fatalf("resolveTemplateBootType should keep explicit boot type, got %q", actual)
	}

	detectedPath := ""
	actual, verified := resolveTemplateBootType("/tmp/windows10-ltsc.qcow2", "windows", "", false, func(path string) string {
		detectedPath = path
		return "uefi"
	})
	if actual != "uefi" || !verified {
		t.Fatalf("resolveTemplateBootType should use detected boot type, got %q", actual)
	}
	if detectedPath != "/tmp/windows10-ltsc.qcow2" {
		t.Fatalf("resolveTemplateBootType detector path = %q, want %q", detectedPath, "/tmp/windows10-ltsc.qcow2")
	}

	actual, verified = resolveTemplateBootType("/tmp/windows10-ltsc.qcow2", "windows", "bios", false, func(string) string {
		return "uefi"
	})
	if actual != "uefi" || !verified {
		t.Fatalf("resolveTemplateBootType should correct stale bios boot type, got %q", actual)
	}

	actual, verified = resolveTemplateBootType("/tmp/fnos.qcow2", "fnos", "bios", false, func(string) string {
		t.Fatal("expected detector to be skipped for explicit non-windows bios template")
		return ""
	})
	if actual != "bios" || !verified {
		t.Fatalf("resolveTemplateBootType should keep explicit bios for non-windows template, got %q", actual)
	}
}

func TestResolveTemplateBootTypeMarksWindowsBIOSAsVerified(t *testing.T) {
	t.Parallel()

	actual, verified := resolveTemplateBootType("/tmp/windows2012r2.qcow2", "windows", "bios", false, func(string) string {
		return "bios"
	})
	if actual != "bios" {
		t.Fatalf("resolveTemplateBootType should keep detected bios boot type, got %q", actual)
	}
	if !verified {
		t.Fatalf("resolveTemplateBootType should mark detected bios boot type as verified")
	}
}

func TestNormalizeLoadedTemplateMetaDetectsMissingBootType(t *testing.T) {
	t.Parallel()

	detectedPath := ""
	meta := normalizeLoadedTemplateMetaWithDetector("Windows10-LTSC", "/tmp/Windows10-LTSC.qcow2", &TemplateMeta{}, true, func(path string) string {
		detectedPath = path
		return "uefi"
	})
	if meta.Type != "windows" {
		t.Fatalf("normalizeLoadedTemplateMeta type = %q, want %q", meta.Type, "windows")
	}
	if meta.BootType != "uefi" {
		t.Fatalf("normalizeLoadedTemplateMeta boot type = %q, want %q", meta.BootType, "uefi")
	}
	if !meta.BootVerified {
		t.Fatalf("normalizeLoadedTemplateMeta should mark detected boot type as verified")
	}
	if meta.Category != "Windows10" {
		t.Fatalf("normalizeLoadedTemplateMeta category = %q, want %q", meta.Category, "Windows10")
	}
	if detectedPath != "/tmp/Windows10-LTSC.qcow2" {
		t.Fatalf("normalizeLoadedTemplateMeta detector path = %q, want %q", detectedPath, "/tmp/Windows10-LTSC.qcow2")
	}
}

func TestNormalizeLoadedTemplateMetaSetsDefaultLinuxCategory(t *testing.T) {
	t.Parallel()

	meta := normalizeLoadedTemplateMetaWithDetector("ubuntu-template", "/tmp/ubuntu-template.qcow2", &TemplateMeta{
		Type: "linux",
	}, true, func(string) string {
		return "bios"
	})

	if meta.Category != "Ubuntu" {
		t.Fatalf("normalizeLoadedTemplateMeta category = %q, want %q", meta.Category, "Ubuntu")
	}
}

func TestDetectBootTypeFromDomainXML(t *testing.T) {
	t.Parallel()

	if actual := detectBootTypeFromDomainXML("<domain><os firmware='efi'></os></domain>"); actual != "uefi" {
		t.Fatalf("detectBootTypeFromDomainXML efi = %q, want %q", actual, "uefi")
	}
	if actual := detectBootTypeFromDomainXML("<domain><os><type>hvm</type></os></domain>"); actual != "bios" {
		t.Fatalf("detectBootTypeFromDomainXML bios = %q, want %q", actual, "bios")
	}
}

func TestExtractDomainNVRAMPath(t *testing.T) {
	t.Parallel()

	xml := `<domain><os firmware='efi'><nvram template='/usr/share/OVMF/OVMF_VARS_4M.ms.fd'>/var/lib/libvirt/qemu/nvram/demo_VARS.fd</nvram></os></domain>`
	if actual := extractDomainNVRAMPath(xml); actual != "/var/lib/libvirt/qemu/nvram/demo_VARS.fd" {
		t.Fatalf("extractDomainNVRAMPath() = %q, want demo NVRAM path", actual)
	}
	if actual := extractDomainNVRAMPath("<domain><os></os></domain>"); actual != "" {
		t.Fatalf("extractDomainNVRAMPath() without nvram = %q, want empty", actual)
	}
}

func TestEnsureDomainNVRAMPath(t *testing.T) {
	t.Parallel()

	path := "/var/lib/libvirt/qemu/nvram/demo_VARS.fd"
	inserted := ensureDomainNVRAMPath("<domain><os firmware='efi'><boot dev='hd'/></os></domain>", path)
	if !strings.Contains(inserted, "<nvram ") || !strings.Contains(inserted, path) || !strings.Contains(inserted, "format='qcow2'") {
		t.Fatalf("ensureDomainNVRAMPath should insert nvram path, got %s", inserted)
	}

	selfClosing := ensureDomainNVRAMPath("<domain><os><nvram template='/tmp/base.fd'/></os></domain>", path)
	if !strings.Contains(selfClosing, "<nvram format='qcow2' template='/tmp/base.fd'>"+path+"</nvram>") {
		t.Fatalf("ensureDomainNVRAMPath should expand self-closing nvram, got %s", selfClosing)
	}

	replaced := ensureDomainNVRAMPath("<domain><os><nvram>/old.fd</nvram></os></domain>", path)
	if strings.Contains(replaced, "/old.fd") || !strings.Contains(replaced, path) || !strings.Contains(replaced, "format='qcow2'") {
		t.Fatalf("ensureDomainNVRAMPath should replace existing nvram path, got %s", replaced)
	}
}

func TestBuildLinuxDiskResizeScriptIncludesFallbacks(t *testing.T) {
	t.Parallel()

	script := buildLinuxDiskResizeScript()
	required := []string{
		"/sys/class/block/$DEV_NAME/partition",
		"basename \"$(dirname \"$SYS_PATH\")\"",
		"growpart",
		"parted -s",
		"sfdisk --no-reread",
		"pvresize",
		"lvextend -r -l +100%FREE",
		"resize2fs",
		"xfs_growfs",
	}
	for _, item := range required {
		if !strings.Contains(script, item) {
			t.Fatalf("expected resize script to include %q", item)
		}
	}
	if strings.Contains(script, "growpart \"$DISK\" \"$PART_NUM\" 2>/dev/null || true") {
		t.Fatalf("resize script should not silently ignore growpart failures")
	}
}

func TestShouldDetectTemplateBootType(t *testing.T) {
	t.Parallel()

	cases := []struct {
		templateType string
		bootType     string
		bootVerified bool
		expected     bool
	}{
		{templateType: "windows", bootType: "", bootVerified: false, expected: true},
		{templateType: "windows", bootType: "bios", bootVerified: false, expected: true},
		{templateType: "windows", bootType: "bios", bootVerified: true, expected: false},
		{templateType: "windows", bootType: "uefi", bootVerified: false, expected: false},
		{templateType: "linux", bootType: "bios", bootVerified: false, expected: false},
		{templateType: "fnos", bootType: "bios", bootVerified: false, expected: false},
	}
	for _, tc := range cases {
		if actual := shouldDetectTemplateBootType(tc.templateType, tc.bootType, tc.bootVerified); actual != tc.expected {
			t.Fatalf("shouldDetectTemplateBootType(%q, %q, %v) = %v, want %v", tc.templateType, tc.bootType, tc.bootVerified, actual, tc.expected)
		}
	}
}

func TestValidateTemplateImportName(t *testing.T) {
	t.Parallel()

	validNames := []string{"demo.tar.gz", "demo.tgz", "demo.qcow2"}
	for _, name := range validNames {
		if err := ValidateTemplateImportName(name); err != nil {
			t.Fatalf("expected valid import name %q, got error: %v", name, err)
		}
	}

	invalidNames := []string{"", "demo.raw", "demo.img", "demo.tar"}
	for _, name := range invalidNames {
		if err := ValidateTemplateImportName(name); err == nil {
			t.Fatalf("expected invalid import name %q to fail", name)
		}
	}
}

func TestResolveImportTemplateSource(t *testing.T) {
	t.Parallel()

	uploadPath := filepath.Join(t.TempDir(), "upload.tar.gz")
	sourcePath, sourceName, cleanupSource, err := resolveImportTemplateSource(&ImportTemplateParams{
		UploadPath: uploadPath,
		UploadName: "upload.tar.gz",
	})
	if err != nil {
		t.Fatalf("expected upload source to resolve, got error: %v", err)
	}
	if sourcePath != uploadPath || sourceName != "upload.tar.gz" || !cleanupSource {
		t.Fatalf("unexpected upload resolution: %q %q %v", sourcePath, sourceName, cleanupSource)
	}

	hostPath := filepath.Join(t.TempDir(), "host.tgz")
	sourcePath, sourceName, cleanupSource, err = resolveImportTemplateSource(&ImportTemplateParams{
		SourcePath: hostPath,
		SourceName: "host.tgz",
	})
	if err != nil {
		t.Fatalf("expected host source to resolve, got error: %v", err)
	}
	if sourcePath != hostPath || sourceName != "host.tgz" || cleanupSource {
		t.Fatalf("unexpected host resolution: %q %q %v", sourcePath, sourceName, cleanupSource)
	}
}

func TestResolveImportTemplateSourceRequiresAbsoluteHostPath(t *testing.T) {
	t.Parallel()

	_, _, _, err := resolveImportTemplateSource(&ImportTemplateParams{
		SourcePath: "relative/demo.tar.gz",
		SourceName: "demo.tar.gz",
	})
	if err == nil {
		t.Fatalf("expected relative host path to fail")
	}
}

func TestGetTemplateExportDir(t *testing.T) {
	previous := config.GlobalConfig
	config.GlobalConfig = &config.Config{TemplateDir: "/var/lib/libvirt/images/templates"}
	defer func() { config.GlobalConfig = previous }()

	got := GetTemplateExportDir()
	want := filepath.Join("/var/lib/libvirt/images/templates", "_exports")
	if got != want {
		t.Fatalf("GetTemplateExportDir() = %q, want %q", got, want)
	}
}

func TestGetTemplateImportTempDir(t *testing.T) {
	previous := config.GlobalConfig
	config.GlobalConfig = &config.Config{TemplateDir: "/var/lib/libvirt/images/templates"}
	defer func() { config.GlobalConfig = previous }()

	got := GetTemplateImportTempDir()
	want := filepath.Join("/var/lib/libvirt/images/templates", "_imports")
	if got != want {
		t.Fatalf("GetTemplateImportTempDir() = %q, want %q", got, want)
	}
}

func TestTemplateExportNamingAndDelete(t *testing.T) {
	tempDir := t.TempDir()
	previous := config.GlobalConfig
	config.GlobalConfig = &config.Config{
		TemplateDir:       "/var/lib/libvirt/images/templates",
		TemplateExportDir: tempDir,
	}
	defer func() { config.GlobalConfig = previous }()

	templateName := "demo-template"
	fileName := GetTemplateExportFileName(templateName)
	if fileName != "demo-template-template-export.tar.gz" {
		t.Fatalf("unexpected export file name: %s", fileName)
	}

	if err := os.WriteFile(GetTemplateExportFilePath(templateName), []byte("disk"), 0o644); err != nil {
		t.Fatalf("write export file: %v", err)
	}
	if !HasExportedTemplate(templateName) {
		t.Fatalf("expected template to be marked as exported")
	}
	if err := DeleteExportedTemplate(templateName); err != nil {
		t.Fatalf("DeleteExportedTemplate failed: %v", err)
	}
	if HasExportedTemplate(templateName) {
		t.Fatalf("expected export file to be deleted")
	}
}

func TestCalculateFileHashes(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "demo.qcow2")
	if err := os.WriteFile(filePath, []byte("template-data"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	hash, err := CalculateFileHashes(filePath)
	if err != nil {
		t.Fatalf("CalculateFileHashes failed: %v", err)
	}
	if hash.FileSize != int64(len("template-data")) {
		t.Fatalf("unexpected file size: %d", hash.FileSize)
	}
	if hash.MD5 == "" || hash.SHA256 == "" {
		t.Fatalf("expected md5 and sha256 to be populated")
	}
}
