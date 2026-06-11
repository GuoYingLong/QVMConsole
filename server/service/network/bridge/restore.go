package bridge

import (
	"fmt"
	"path/filepath"
	"strings"

	"kvm_console/utils"
)

func writeBridgeRestoreScript(bridge, uplink string, migrateHostIP bool) error {
	content := buildBridgeRestoreScriptContent(bridge, uplink, migrateHostIP)
	_, err := HookWriteFileIfChanged(bridgeRestoreScriptPath(bridge), []byte(content), 0755)
	return err
}

func buildBridgeRestoreScriptContent(bridge, uplink string, migrateHostIP bool) string {
	content := fmt.Sprintf(`#!/bin/bash
set -e
BRIDGE=%s
UPLINK=%s
`, utils.ShellSingleQuote(bridge), utils.ShellSingleQuote(uplink))
	if migrateHostIP {
		content += `# 先记录物理口当前 DHCP/静态地址，加入 OVS 后再迁移到 bridge。
`
		content += bridgeHostIPCaptureShell()
	}
	content += `ovs-vsctl --may-exist add-br "$BRIDGE"
ip link set "$BRIDGE" up
ovs-vsctl --may-exist add-port "$BRIDGE" "$UPLINK"
ip link set "$UPLINK" up
`
	if migrateHostIP {
		content += bridgeHostIPApplyShell()
		content += `# DNS 迁移到 bridge，避免默认路由切换后解析仍绑定在物理口。
`
		content += bridgeResolvedDNSShell()
	}
	return content
}

func bridgeRestoreScriptPath(bridge string) string {
	return filepath.Join(bridgeConfigDir, strings.TrimSpace(bridge)+".sh")
}
