package service

import (
	"fmt"
	"strings"

	"kvm_console/service/vm_xml"
	"kvm_console/utils"
)

// SetVMBootType 修改虚拟机引导方式。该操作要求虚拟机关机后执行。
func SetVMBootType(name, bootType string) error {
	normalized := vm_xml.NormalizeVMBootType(bootType)
	if normalized == "" {
		return fmt.Errorf("不支持的引导方式: %s", bootType)
	}

	stateResult := utils.ExecCommand("virsh", "domstate", name)
	if stateResult.Error != nil {
		return fmt.Errorf("获取虚拟机状态失败: %s", stateResult.Stderr)
	}
	state := strings.TrimSpace(stateResult.Stdout)
	if state == "running" || state == "paused" {
		return fmt.Errorf("请先关机后再修改引导方式")
	}

	xmlResult := utils.ExecCommand("virsh", "dumpxml", name, "--inactive")
	if xmlResult.Error != nil {
		return fmt.Errorf("获取虚拟机 XML 失败: %s", xmlResult.Stderr)
	}

	currentBootType := vm_xml.ParseVMBootTypeFromDomainXML(xmlResult.Stdout)
	if currentBootType == normalized {
		return nil
	}
	if err := vm_xml.EnsureVMUEFINVRAMFile(name, xmlResult.Stdout, normalized); err != nil {
		return err
	}

	updatedXML, err := vm_xml.ApplyVMBootTypeToDomainXML(name, xmlResult.Stdout, normalized)
	if err != nil {
		return err
	}
	if err := SetVMInactiveDomainXML(name, updatedXML); err != nil {
		return fmt.Errorf("设置引导方式失败: %w", err)
	}
	return nil
}
