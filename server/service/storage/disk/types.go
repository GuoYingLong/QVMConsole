package disk

import "strings"

// ExtraDiskParam defines extra disk parameters when creating a VM.
// Moved from service root vm_create.go to avoid circular dependency.
type ExtraDiskParam struct {
	Size          int    `json:"size"`            // GB
	Format        string `json:"format"`          // qcow2/raw
	Bus           string `json:"bus"`             // disk bus: virtio/scsi/sata/ide
	StoragePoolID string `json:"storage_pool_id"` // storage pool for extra disk
	// IOPS limits (admin only, 0 = unlimited)
	IOPSTotal int `json:"iops_total,omitempty"`
	IOPSRead  int `json:"iops_read,omitempty"`
	IOPSWrite int `json:"iops_write,omitempty"`
}

// NormalizeVMDiskBus normalizes a disk bus string to a known value.
// Moved from service root lightweight_vm_registration.go to avoid circular dependency.
func NormalizeVMDiskBus(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "":
		return ""
	case "scsi":
		return "scsi"
	case "sata":
		return "sata"
	case "ide":
		return "ide"
	default:
		return "virtio"
	}
}
