package bridge

import (
	"fmt"

	"kvm_console/utils"
)

func migrateInterfaceIPv4ToBridge(uplink, bridge string) {
	script := fmt.Sprintf(`set -e
UPLINK=%s
BRIDGE=%s
%s
`, utils.ShellSingleQuote(uplink), utils.ShellSingleQuote(bridge), bridgeHostIPMigrationShell())
	utils.ExecCommand("bash", "-c", script)
}

func migrateBridgeIPv4ToInterface(bridge, uplink string) {
	script := fmt.Sprintf(`set -e
BRIDGE=%s
UPLINK=%s
%s
`, utils.ShellSingleQuote(bridge), utils.ShellSingleQuote(uplink), bridgeHostIPRollbackShell())
	utils.ExecCommand("bash", "-c", script)
}

func bridgeHostIPMigrationShell() string {
	return bridgeHostIPCaptureShell() + bridgeHostIPApplyShell()
}

func bridgeHostIPRollbackShell() string {
	return bridgeHostIPCaptureFromBridgeShell() + bridgeHostIPApplyToUplinkShell()
}

func bridgeHostIPCaptureShell() string {
	return `HOST_ADDRS="$(ip -4 -o addr show dev "$UPLINK" scope global 2>/dev/null | awk '{print $4}')"
HOST_GW="$(ip -4 route show default dev "$UPLINK" 2>/dev/null | awk '{print $3; exit}')"
HOST_METRIC="$(ip -4 route show default dev "$UPLINK" 2>/dev/null | awk '{for (i=1;i<=NF;i++) if ($i=="metric") {print $(i+1); exit}}')"
`
}

func bridgeHostIPCaptureFromBridgeShell() string {
	return `HOST_ADDRS="$(ip -4 -o addr show dev "$BRIDGE" scope global 2>/dev/null | awk '{print $4}')"
HOST_GW="$(ip -4 route show default dev "$BRIDGE" 2>/dev/null | awk '{print $3; exit}')"
HOST_METRIC="$(ip -4 route show default dev "$BRIDGE" 2>/dev/null | awk '{for (i=1;i<=NF;i++) if ($i=="metric") {print $(i+1); exit}}')"
`
}

func bridgeHostIPApplyShell() string {
	return `if [ -n "$HOST_ADDRS" ]; then
  ip addr flush dev "$UPLINK"
  while IFS= read -r addr; do
    [ -n "$addr" ] || continue
    ip addr replace "$addr" dev "$BRIDGE"
  done <<< "$HOST_ADDRS"
fi
if [ -n "$HOST_GW" ]; then
  ip route del "$HOST_GW" dev "$UPLINK" 2>/dev/null || true
  ip route replace "$HOST_GW" dev "$BRIDGE" scope link
  if [ -n "$HOST_METRIC" ]; then
    ip route replace default via "$HOST_GW" dev "$BRIDGE" metric "$HOST_METRIC"
  else
    ip route replace default via "$HOST_GW" dev "$BRIDGE"
  fi
fi
`
}

func bridgeHostIPApplyToUplinkShell() string {
	return `ip link set "$UPLINK" up
if [ -n "$HOST_ADDRS" ]; then
  ip addr flush dev "$BRIDGE"
  while IFS= read -r addr; do
    [ -n "$addr" ] || continue
    ip addr replace "$addr" dev "$UPLINK"
  done <<< "$HOST_ADDRS"
fi
if [ -n "$HOST_GW" ]; then
  ip route del "$HOST_GW" dev "$BRIDGE" 2>/dev/null || true
  ip route replace "$HOST_GW" dev "$UPLINK" scope link
  if [ -n "$HOST_METRIC" ]; then
    ip route replace default via "$HOST_GW" dev "$UPLINK" metric "$HOST_METRIC"
  else
    ip route replace default via "$HOST_GW" dev "$UPLINK"
  fi
fi
`
}
