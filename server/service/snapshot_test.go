package service

import (
	"strings"
	"testing"
)

func TestValidateSnapshotName(t *testing.T) {
	t.Parallel()

	valid := []string{"snap1", "snap_1", "snap-1", "snap.1", "_snap"}
	for _, name := range valid {
		if err := ValidateSnapshotName(name); err != nil {
			t.Fatalf("expected snapshot name %q to be valid, got %v", name, err)
		}
	}

	invalid := []string{"", "测试", "snap 1", "-snap", ".snap", "snap/1"}
	for _, name := range invalid {
		if err := ValidateSnapshotName(name); err == nil {
			t.Fatalf("expected snapshot name %q to be rejected", name)
		}
	}
}

func TestNormalizeSnapshotNameGeneratesName(t *testing.T) {
	t.Parallel()

	name, err := NormalizeSnapshotName("")
	if err != nil {
		t.Fatalf("NormalizeSnapshotName returned error: %v", err)
	}
	if err := ValidateSnapshotName(name); err != nil {
		t.Fatalf("generated snapshot name should be valid, got %q: %v", name, err)
	}
	if !strings.HasPrefix(name, "snap_") {
		t.Fatalf("generated snapshot name should use snap_ prefix, got %q", name)
	}
}

func TestFormatSnapshotCreateError(t *testing.T) {
	t.Parallel()

	err := formatSnapshotCreateError("error: internal error: unable to execute QEMU command 'snapshot-save': Invalid job ID 'internal-snapshot-save-测试'")
	if err == nil || !strings.Contains(err.Error(), "快照名称") {
		t.Fatalf("expected invalid job id to be translated, got %v", err)
	}

	err = formatSnapshotCreateError("error: Requested operation is not valid: cannot migrate domain: Migration is disabled when VirtFS export path '/var/lib/kvm-user-storage/admin/iso' is mounted in the guest using mount_tag 'user_admin_iso'")
	if err == nil || !strings.Contains(err.Error(), "VirtFS") {
		t.Fatalf("expected VirtFS migration error to be translated, got %v", err)
	}
}

func TestParseSnapshotInfoOutput(t *testing.T) {
	t.Parallel()

	info := parseSnapshotInfoOutput(`
Name:           snap_20260504_182455_2b4270d7
Domain:         vmuyprjq4s
Current:        no
State:          running
Location:       internal
Parent:         -
Children:       1
Descendants:    2
Metadata:       yes
`)

	if info.State != "running" || info.Location != "internal" || info.Children != 1 || info.Descendants != 2 {
		t.Fatalf("unexpected snapshot info: %#v", info)
	}
}

func TestFormatSnapshotDeleteError(t *testing.T) {
	t.Parallel()

	err := formatSnapshotDeleteError("error: Operation not supported: disk image 'vda' for internal snapshot 'snap' is not the same as disk image currently used by VM")
	if err == nil || !strings.Contains(err.Error(), "磁盘与目标内部快照所在磁盘不一致") {
		t.Fatalf("expected internal snapshot disk mismatch to be translated, got %v", err)
	}
}

func TestIsLikelyExternalSnapshotOverlay(t *testing.T) {
	t.Parallel()

	if !isLikelyExternalSnapshotOverlay("/var/lib/libvirt/images/vm.snap_20260504_182557_8cd33a2f") {
		t.Fatalf("expected generated system disk snapshot overlay to be detected")
	}
	if !isLikelyExternalSnapshotOverlay("/var/lib/kvm-storage/sdb/vm-disks/vm-vdb.snap_20260504_182557_8cd33a2f") {
		t.Fatalf("expected generated data disk snapshot overlay to be detected")
	}
	if isLikelyExternalSnapshotOverlay("/var/lib/libvirt/images/vm.qcow2") {
		t.Fatalf("expected normal VM disk not to be treated as snapshot overlay")
	}
}

func TestGenerateStandaloneDiskPathUsesQcow2Suffix(t *testing.T) {
	t.Parallel()

	got := generateStandaloneDiskPath("/var/lib/libvirt/images/vm.snap_20260504_184903_3f666fb1")
	if !strings.HasPrefix(got, "/var/lib/libvirt/images/vm.snap_20260504_184903_3f666fb1.consolidated_") {
		t.Fatalf("unexpected consolidated path prefix: %q", got)
	}
	if !strings.HasSuffix(got, ".qcow2") {
		t.Fatalf("consolidated path should use qcow2 suffix: %q", got)
	}
}

func TestGenerateExternalSnapshotRestoreOverlayPathIsRecognized(t *testing.T) {
	t.Parallel()

	got := generateExternalSnapshotRestoreOverlayPath("/var/lib/libvirt/images/vm.qcow2", "vm/test", "vda", "snap_20260504_195233_0b93c473")
	if !strings.HasPrefix(got, "/var/lib/libvirt/images/vm.snap_restore_vm_test_vda_snap_20260504_195233_0b93c473_") {
		t.Fatalf("unexpected restore overlay prefix: %q", got)
	}
	if !strings.HasSuffix(got, ".qcow2") {
		t.Fatalf("restore overlay should use qcow2 suffix: %q", got)
	}
	if !isLikelyExternalSnapshotOverlay(got) {
		t.Fatalf("restore overlay should be recognized as managed external overlay: %q", got)
	}
}

func TestIsManagedSnapshotResidualFileName(t *testing.T) {
	t.Parallel()

	if !isManagedSnapshotResidualFileName("vmuyprjq4s", "snap_20260504_195233_0b93c473", "vmuyprjq4s.snap_20260504_195233_0b93c473", false) {
		t.Fatalf("expected matching system disk snapshot residual to be detected")
	}
	if !isManagedSnapshotResidualFileName("vmuyprjq4s", "snap_20260504_195233_0b93c473", "vmuyprjq4s-vdb.snap_restore_vmuyprjq4s_vdb_snap_20260504_195233_0b93c473_abcd.qcow2", false) {
		t.Fatalf("expected matching restore overlay residual to be detected")
	}
	if isManagedSnapshotResidualFileName("vmuyprjq4s", "snap_20260504_195233_0b93c473", "vmuyprjq4s.snap_20260504_200923_5761eea5", false) {
		t.Fatalf("single snapshot cleanup should not match another snapshot")
	}
	if !isManagedSnapshotResidualFileName("vmuyprjq4s", "", "vmuyprjq4s.snap_20260504_200923_5761eea5", true) {
		t.Fatalf("delete-all cleanup should match any snapshot residual for the VM")
	}
	if isManagedSnapshotResidualFileName("vmuyprjq4s", "", "vmuyprjq4s.qcow2", true) {
		t.Fatalf("normal VM disk should not be treated as snapshot residual")
	}
}

func TestExtractSourceFilePathsFromXML(t *testing.T) {
	t.Parallel()

	files := extractSourceFilePathsFromXML(`
<domain>
  <devices>
    <disk><source file='/var/lib/libvirt/images/vm.qcow2'/></disk>
    <disk><source file="/var/lib/kvm-storage/sdb/vm-disks/vm-vdb.snap_20260504"/></disk>
  </devices>
</domain>`)

	if len(files) != 2 {
		t.Fatalf("expected 2 source files, got %#v", files)
	}
	if files[0] != "/var/lib/libvirt/images/vm.qcow2" || files[1] != "/var/lib/kvm-storage/sdb/vm-disks/vm-vdb.snap_20260504" {
		t.Fatalf("unexpected source files: %#v", files)
	}
}

func TestParseExternalSnapshotDiskFilesOnlyUsesTopLevelDisks(t *testing.T) {
	t.Parallel()

	snapshotXML := `
<domainsnapshot>
  <name>snap_20260504_083828_541d6e9c</name>
  <disks>
    <disk name='vda' snapshot='external' type='file'>
      <source file='/var/lib/libvirt/images/vm.snap_0838'/>
    </disk>
    <disk name='vdb' snapshot='external' type='file'>
      <source file='/var/lib/libvirt/images/vm-vdb.snap_0838'/>
    </disk>
  </disks>
  <domain type='kvm'>
    <devices>
      <disk type='file' device='disk'>
        <source file='/var/lib/libvirt/images/vm.snap_0739'/>
        <backingStore type='file'>
          <source file='/var/lib/libvirt/images/vm.snap_0558'/>
        </backingStore>
      </disk>
      <disk type='file' device='disk'>
        <source file='/var/lib/libvirt/images/vm-vdb.snap_0739'/>
        <backingStore type='file'>
          <source file='/var/lib/libvirt/images/vm-vdb.snap_0558'/>
        </backingStore>
      </disk>
    </devices>
  </domain>
  <inactiveDomain type='kvm'>
    <devices>
      <disk type='file' device='disk'>
        <source file='/var/lib/libvirt/images/vm.snap_0739'/>
      </disk>
    </devices>
  </inactiveDomain>
</domainsnapshot>`

	files, err := parseExternalSnapshotDiskFiles(snapshotXML)
	if err != nil {
		t.Fatalf("parseExternalSnapshotDiskFiles returned error: %v", err)
	}

	want := []string{
		"/var/lib/libvirt/images/vm.snap_0838",
		"/var/lib/libvirt/images/vm-vdb.snap_0838",
	}
	if len(files) != len(want) {
		t.Fatalf("expected %d files, got %d: %#v", len(want), len(files), files)
	}
	for i := range want {
		if files[i] != want[i] {
			t.Fatalf("file %d mismatch: got %q want %q", i, files[i], want[i])
		}
	}
}

func TestParseExternalSnapshotOriginalDiskFilesMapsByTarget(t *testing.T) {
	t.Parallel()

	snapshotXML := `
<domainsnapshot>
  <disks>
    <disk name='vda' snapshot='external' type='file'>
      <source file='/var/lib/libvirt/images/vm.snap_0838'/>
    </disk>
    <disk name='vdb' snapshot='external' type='file'>
      <source file='/var/lib/libvirt/images/vm-vdb.snap_0838'/>
    </disk>
  </disks>
  <domain type='kvm'>
    <devices>
      <disk type='file' device='disk'>
        <source file='/var/lib/libvirt/images/vm.snap_0739'/>
        <backingStore type='file'>
          <source file='/var/lib/libvirt/images/vm.snap_0558'/>
        </backingStore>
        <target dev='vda' bus='virtio'/>
      </disk>
      <disk type='file' device='disk'>
        <source file='/var/lib/libvirt/images/vm-vdb.snap_0739'/>
        <backingStore type='file'>
          <source file='/var/lib/libvirt/images/vm-vdb.snap_0558'/>
        </backingStore>
        <target dev='vdb' bus='virtio'/>
      </disk>
    </devices>
  </domain>
</domainsnapshot>`

	files, err := parseExternalSnapshotOriginalDiskFiles(snapshotXML)
	if err != nil {
		t.Fatalf("parseExternalSnapshotOriginalDiskFiles returned error: %v", err)
	}

	if files["vda"] != "/var/lib/libvirt/images/vm.snap_0739" {
		t.Fatalf("vda source mismatch: %#v", files)
	}
	if files["vdb"] != "/var/lib/libvirt/images/vm-vdb.snap_0739" {
		t.Fatalf("vdb source mismatch: %#v", files)
	}
	for _, leaked := range []string{"/var/lib/libvirt/images/vm.snap_0558", "/var/lib/libvirt/images/vm-vdb.snap_0558"} {
		for _, got := range files {
			if got == leaked {
				t.Fatalf("backing source %q should not be used as original disk: %#v", leaked, files)
			}
		}
	}
}

func TestIsManagedHostStoragePath(t *testing.T) {
	t.Parallel()

	if !isManagedHostStoragePath("/var/lib/kvm-storage/sdb/vm-disks/vm-vdb.snap_20260504") {
		t.Fatalf("expected kvm-storage path to be detected")
	}
	if isManagedHostStoragePath("/var/lib/kvm-storage-other/sdb/vm.qcow2") {
		t.Fatalf("expected sibling path not to be detected")
	}
	if isManagedHostStoragePath("/var/lib/libvirt/images/vm.qcow2") {
		t.Fatalf("expected default libvirt image path not to be detected")
	}
}

func TestUpsertManagedBlock(t *testing.T) {
	t.Parallel()

	block := buildManagedAppArmorBlock([]string{"/var/lib/kvm-storage/** rwk,"})
	updated := upsertManagedBlock("existing rule\n", block)
	if !strings.Contains(updated, appArmorManagedBlockBegin) {
		t.Fatalf("expected managed block to be appended: %q", updated)
	}

	replacement := buildManagedAppArmorBlock([]string{"/var/lib/kvm-storage/** r,"})
	replaced := upsertManagedBlock(updated, replacement)
	if !strings.Contains(replaced, "/var/lib/kvm-storage/** r,") || strings.Contains(replaced, "/var/lib/kvm-storage/** rwk,") {
		t.Fatalf("expected managed block to be replaced: %q", replaced)
	}
}
