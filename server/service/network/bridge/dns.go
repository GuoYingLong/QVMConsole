package bridge

import (
	"net"
	"strings"

	"kvm_console/utils"
)

func ensureBridgeResolvedDNS(uplink, bridge string) {
	uplink = strings.TrimSpace(uplink)
	bridge = strings.TrimSpace(bridge)
	if uplink == "" || bridge == "" {
		return
	}
	if result := utils.ExecCommand("bash", "-c", "command -v resolvectl"); result.Error != nil {
		return
	}
	servers := resolvectlDNSServers(uplink)
	if len(servers) == 0 {
		servers = resolvectlDNSServers("")
	}
	if len(servers) > 0 {
		args := append([]string{"dns", bridge}, servers...)
		utils.ExecCommand("resolvectl", args...)
	}
	utils.ExecCommand("resolvectl", "default-route", bridge, "yes")
	utils.ExecCommand("resolvectl", "domain", bridge, "~.")
}

func resolvectlDNSServers(link string) []string {
	args := []string{"dns"}
	if strings.TrimSpace(link) != "" {
		args = append(args, strings.TrimSpace(link))
	}
	result := utils.ExecCommand("resolvectl", args...)
	if result.Error != nil {
		return nil
	}
	return parseResolvectlDNSServers(result.Stdout)
}

func parseResolvectlDNSServers(text string) []string {
	seen := map[string]bool{}
	var servers []string
	for _, field := range strings.Fields(text) {
		value := strings.Trim(field, ",;")
		if strings.HasPrefix(value, "[") || strings.HasSuffix(value, "]") || strings.Contains(value, "(") || strings.Contains(value, ")") {
			continue
		}
		host, _, splitErr := net.SplitHostPort(value)
		if splitErr == nil {
			value = host
		}
		ip := net.ParseIP(value)
		if ip == nil || seen[value] {
			continue
		}
		seen[value] = true
		servers = append(servers, value)
	}
	return servers
}

func bridgeResolvedDNSShell() string {
	return `if command -v resolvectl >/dev/null 2>&1; then
  DNS_SERVERS="$(resolvectl dns "$UPLINK" 2>/dev/null | sed 's/.*://' | xargs)"
  if [ -z "$DNS_SERVERS" ]; then
    DNS_SERVERS="$(resolvectl dns 2>/dev/null | sed 's/.*://' | xargs)"
  fi
  if [ -n "$DNS_SERVERS" ]; then
    resolvectl dns "$BRIDGE" $DNS_SERVERS || true
  fi
  resolvectl default-route "$BRIDGE" yes || true
  resolvectl domain "$BRIDGE" '~.' || true
fi
`
}
