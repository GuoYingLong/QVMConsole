package host

import (
	"context"
	"time"

	"kvm_console/model"
)

var (
	HookRemoteSSHExec                    func(ctx context.Context, node model.HostNode, command string, timeout time.Duration, tolerateRemoteExit bool) (string, error)
	HookCallNodeAPI                      func(node model.HostNode, method, path string, body interface{}, out interface{}) ([]byte, error)
	HookShutdownVM                       func(name string) error
	HookDestroyVM                        func(name string) error
	HookWaitVMShutdownForDisable         func(vmName string, timeout time.Duration) bool
	HookClearRuntimeCachesForMaintenance func()
)
