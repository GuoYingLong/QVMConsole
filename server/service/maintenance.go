package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"kvm_console/config"
	"kvm_console/taskqueue"
	"kvm_console/utils"
)

// MaintenanceModeTaskParams 维护模式任务参数。
type MaintenanceModeTaskParams struct {
	ServiceUnits []string `json:"service_units,omitempty"`
}

// MaintenanceModeTaskResult 维护模式任务执行结果。
type MaintenanceModeTaskResult struct {
	StoppedVMs       []string `json:"stopped_vms,omitempty"`
	DisabledServices []string `json:"disabled_services,omitempty"`
	EnabledServices  []string `json:"enabled_services,omitempty"`
	Warnings         []string `json:"warnings,omitempty"`
}

// IsMaintenanceModeEnabled 判断当前是否已开启维护模式。
func IsMaintenanceModeEnabled() bool {
	return config.GlobalConfig != nil && config.GlobalConfig.MaintenanceMode
}

// EnsureMaintenanceModeDisabled 校验维护模式是否允许当前操作。
func EnsureMaintenanceModeDisabled(action string) error {
	if !IsMaintenanceModeEnabled() {
		return nil
	}
	action = strings.TrimSpace(action)
	if action == "" {
		action = "执行该操作"
	}
	return fmt.Errorf("系统当前处于维护模式，暂不允许%s，请先关闭维护模式", action)
}

// IsMaintenanceModeError 判断错误是否由维护模式拦截导致。
func IsMaintenanceModeError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "系统当前处于维护模式")
}

// IsLibvirtUnavailableError 判断是否为 libvirt 服务当前不可用导致的错误。
func IsLibvirtUnavailableError(err error) bool {
	if err == nil {
		return false
	}
	return isLibvirtUnavailableText(err.Error())
}

func isLibvirtUnavailableText(text string) bool {
	text = strings.ToLower(strings.TrimSpace(text))
	return strings.Contains(text, "failed to connect to the hypervisor") ||
		strings.Contains(text, "failed to connect socket") ||
		strings.Contains(text, "libvirt-sock") ||
		strings.Contains(text, "libvirt socket") ||
		strings.Contains(text, "no such file or directory")
}

// ParseMaintenanceServiceUnits 解析维护模式 service units 配置。
func ParseMaintenanceServiceUnits(raw string) []string {
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == '\n' || r == '\r' || r == ';' || r == '\t'
	})
	seen := make(map[string]bool)
	units := make([]string, 0, len(parts))
	for _, part := range parts {
		unit := strings.TrimSpace(part)
		if unit == "" || seen[unit] {
			continue
		}
		seen[unit] = true
		units = append(units, unit)
	}
	return units
}

func resolveMaintenanceServiceUnits(params *MaintenanceModeTaskParams) []string {
	if params != nil && len(params.ServiceUnits) > 0 {
		return ParseMaintenanceServiceUnits(strings.Join(params.ServiceUnits, ","))
	}
	if config.GlobalConfig == nil {
		return nil
	}
	return ParseMaintenanceServiceUnits(config.GlobalConfig.MaintenanceServiceUnits)
}

// EnterMaintenanceMode 执行维护模式收敛流程。
func EnterMaintenanceMode(ctx context.Context, params *MaintenanceModeTaskParams, progressFn func(int, string)) (*MaintenanceModeTaskResult, error) {
	if progressFn == nil {
		progressFn = func(int, string) {}
	}

	result := &MaintenanceModeTaskResult{}
	progressFn(5, "正在收集维护模式执行信息...")

	shutdownTimeout := 40
	if config.GlobalConfig != nil && config.GlobalConfig.MaintenanceVMShutdownTimeoutSeconds > 0 {
		shutdownTimeout = config.GlobalConfig.MaintenanceVMShutdownTimeoutSeconds
	}

	progressFn(20, "正在关闭运行中的虚拟机...")
	stoppedVMs, warnings, err := stopAllRunningVMsForMaintenance(ctx, time.Duration(shutdownTimeout)*time.Second)
	if err != nil {
		return result, err
	}
	result.StoppedVMs = stoppedVMs
	result.Warnings = append(result.Warnings, warnings...)

	progressFn(70, "正在停用宿主机相关服务...")
	disabledServices, warnings, err := disableMaintenanceServiceUnits(ctx, resolveMaintenanceServiceUnits(params))
	if err != nil {
		return result, err
	}
	result.DisabledServices = disabledServices
	result.Warnings = append(result.Warnings, warnings...)

	clearRuntimeCachesForMaintenance()
	progressFn(100, "维护模式已启用，启动类操作已被阻止")
	return result, nil
}

// ExitMaintenanceMode 恢复维护模式期间停用的服务。
func ExitMaintenanceMode(ctx context.Context, params *MaintenanceModeTaskParams, progressFn func(int, string)) (*MaintenanceModeTaskResult, error) {
	if progressFn == nil {
		progressFn = func(int, string) {}
	}

	result := &MaintenanceModeTaskResult{}
	progressFn(15, "正在恢复宿主机相关服务...")

	enabledServices, warnings, err := enableMaintenanceServiceUnits(ctx, resolveMaintenanceServiceUnits(params))
	if err != nil {
		return result, err
	}
	result.EnabledServices = enabledServices
	result.Warnings = append(result.Warnings, warnings...)

	progressFn(100, "维护模式已关闭，宿主机相关服务已恢复")
	return result, nil
}

func stopAllRunningVMsForMaintenance(ctx context.Context, timeout time.Duration) ([]string, []string, error) {
	result := utils.ExecShell("virsh list --name --state-running 2>/dev/null | grep -v '^$'")
	if result.Error != nil || strings.TrimSpace(result.Stdout) == "" {
		return nil, nil, nil
	}

	var stopped []string
	var warnings []string
	for _, vmName := range strings.Split(strings.TrimSpace(result.Stdout), "\n") {
		if err := checkMaintenanceCanceled(ctx); err != nil {
			return stopped, warnings, err
		}

		vmName = strings.TrimSpace(vmName)
		if vmName == "" {
			continue
		}

		state := strings.ToLower(strings.TrimSpace(utils.ExecCommand("virsh", "domstate", vmName).Stdout))
		needForceOff := strings.Contains(state, "paused")
		if !needForceOff {
			if err := ShutdownVM(vmName); err != nil {
				needForceOff = true
			} else if !waitVMShutdownForDisable(vmName, timeout) {
				needForceOff = true
			}
		}
		if needForceOff {
			if err := DestroyVM(vmName); err != nil {
				warnings = append(warnings, fmt.Sprintf("关闭虚拟机 %s 失败: %s", vmName, err.Error()))
				continue
			}
		}
		stopped = append(stopped, vmName)
	}
	return stopped, warnings, nil
}

func disableMaintenanceServiceUnits(ctx context.Context, units []string) ([]string, []string, error) {
	var applied []string
	var warnings []string

	panelUnit := ""
	if config.GlobalConfig != nil {
		panelUnit = strings.TrimSpace(config.GlobalConfig.ServiceUnitName)
	}

	for _, unit := range units {
		if err := checkMaintenanceCanceled(ctx); err != nil {
			return applied, warnings, err
		}
		if unit == "" {
			continue
		}

		if panelUnit != "" && unit == panelUnit {
			enableResult := utils.ExecCommand("systemctl", "enable", unit)
			if enableResult.Error != nil {
				warnings = append(warnings, fmt.Sprintf("确保面板服务 %s 开机自启失败: %s", unit, firstNonEmpty(enableResult.Stderr, enableResult.Error.Error())))
			} else {
				warnings = append(warnings, fmt.Sprintf("已跳过面板服务 %s，面板始终保持开机自启", unit))
			}
			continue
		}

		disableResult := utils.ExecCommand("systemctl", "disable", unit)
		if disableResult.Error != nil {
			warnings = append(warnings, fmt.Sprintf("禁用服务 %s 失败: %s", unit, firstNonEmpty(disableResult.Stderr, disableResult.Error.Error())))
			continue
		}

		stopResult := utils.ExecCommand("systemctl", "stop", unit)
		if stopResult.Error != nil {
			warnings = append(warnings, fmt.Sprintf("停止服务 %s 失败: %s", unit, firstNonEmpty(stopResult.Stderr, stopResult.Error.Error())))
			continue
		}
		applied = append(applied, unit)
	}

	return applied, warnings, nil
}

func enableMaintenanceServiceUnits(ctx context.Context, units []string) ([]string, []string, error) {
	var applied []string
	var warnings []string
	panelUnit := ""
	if config.GlobalConfig != nil {
		panelUnit = strings.TrimSpace(config.GlobalConfig.ServiceUnitName)
	}

	for _, unit := range units {
		if err := checkMaintenanceCanceled(ctx); err != nil {
			return applied, warnings, err
		}
		if unit == "" {
			continue
		}

		if panelUnit != "" && unit == panelUnit {
			enableResult := utils.ExecCommand("systemctl", "enable", unit)
			if enableResult.Error != nil {
				warnings = append(warnings, fmt.Sprintf("确保面板服务 %s 开机自启失败: %s", unit, firstNonEmpty(enableResult.Stderr, enableResult.Error.Error())))
			}
			continue
		}

		enableResult := utils.ExecCommand("systemctl", "enable", unit)
		if enableResult.Error != nil {
			warnings = append(warnings, fmt.Sprintf("启用服务 %s 失败: %s", unit, firstNonEmpty(enableResult.Stderr, enableResult.Error.Error())))
			continue
		}

		startResult := utils.ExecCommand("systemctl", "start", unit)
		if startResult.Error != nil {
			warnings = append(warnings, fmt.Sprintf("启动服务 %s 失败: %s", unit, firstNonEmpty(startResult.Stderr, startResult.Error.Error())))
			continue
		}

		applied = append(applied, unit)
	}

	return applied, warnings, nil
}

func clearRuntimeCachesForMaintenance() {
	statsCache.Lock()
	statsCache.data = make(map[string]*VmStats)
	statsCache.Unlock()
}

func checkMaintenanceCanceled(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return taskqueue.ErrTaskCanceled
	default:
		return nil
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
