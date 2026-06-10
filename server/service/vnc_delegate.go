package service

// VNC function delegates - forward to service/vnc subpackage
// Maintains backward compatibility for callers using service.XXX()

import vncpkg "kvm_console/service/vnc"

// ── Exported delegates ──

func GetVncStatus(vmName string) (*VncInfo, error) {
	return vncpkg.GetVncStatus(vmName)
}

func EnableVnc(vmName, password string) error {
	return vncpkg.EnableVnc(vmName, password)
}

func DisableVnc(vmName string) error {
	return vncpkg.DisableVnc(vmName)
}

func ChangeVncPassword(vmName, newPassword string) error {
	return vncpkg.ChangeVncPassword(vmName, newPassword)
}

func GetVncConnInfo(vmName string) (*VncConnInfo, error) {
	return vncpkg.GetVncConnInfo(vmName)
}

func ExposeVnc(vmName string, expose bool) error {
	return vncpkg.ExposeVnc(vmName, expose)
}
