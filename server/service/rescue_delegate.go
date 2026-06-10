package service

// Rescue function delegates - forward to service/rescue subpackage
// Maintains backward compatibility for callers using service.XXX()

import rescuepkg "kvm_console/service/rescue"

// ── Exported delegates ──

func StartRescue(vmName, rescueISO string, progress func(int, string)) error {
	return rescuepkg.StartRescue(vmName, rescueISO, progress)
}

func StopRescue(vmName string, progress func(int, string)) error {
	return rescuepkg.StopRescue(vmName, progress)
}

func IsInRescueMode(vmName string) bool {
	return rescuepkg.IsInRescueMode(vmName)
}
