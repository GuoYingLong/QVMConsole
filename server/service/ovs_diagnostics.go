package service

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net"
	"regexp"
	"sort"
	"strings"

	"kvm_console/utils"
)

type OVSStatus struct {
	Bridge             string              `json:"bridge"`
	GatewayIP          string              `json:"gateway_ip"`
	SubnetCIDR         string              `json:"subnet_cidr"`
	Uplink             string              `json:"uplink"`
	BridgeExists       bool                `json:"bridge_exists"`
	BridgeHasGateway   bool                `json:"bridge_has_gateway"`
	OpenVSwitchService OVSServiceStatus    `json:"openvswitch_service"`
	DNSMasqService     OVSServiceStatus    `json:"dnsmasq_service"`
	IPForwardEnabled   bool                `json:"ip_forward_enabled"`
	NATRule            OVSRuleStatus       `json:"nat_rule"`
	ForwardOutRule     OVSRuleStatus       `json:"forward_out_rule"`
	ForwardReturnRule  OVSRuleStatus       `json:"forward_return_rule"`
	Healthy            bool                `json:"healthy"`
	Issues             []string            `json:"issues"`
	RepairSuggestions  []string            `json:"repair_suggestions"`
	RawCommandErrors   []OVSCommandFailure `json:"raw_command_errors,omitempty"`
}

type OVSServiceStatus struct {
	Name   string `json:"name"`
	Active bool   `json:"active"`
	State  string `json:"state"`
	Error  string `json:"error,omitempty"`
}

type OVSRuleStatus struct {
	Name    string `json:"name"`
	Command string `json:"command"`
	Exists  bool   `json:"exists"`
	Error   string `json:"error,omitempty"`
}

type OVSCommandFailure struct {
	Command string `json:"command"`
	Error   string `json:"error"`
}

type OVSPortList struct {
	Bridge string          `json:"bridge"`
	Ports  []OVSPortStatus `json:"ports"`
	Issues []string        `json:"issues"`
}

type OVSPortStatus struct {
	Name     string   `json:"name"`
	OFPort   string   `json:"ofport"`
	Type     string   `json:"type"`
	VMName   string   `json:"vm_name"`
	MAC      string   `json:"mac"`
	IP       string   `json:"ip"`
	IPSource string   `json:"ip_source"`
	Issues   []string `json:"issues"`
}

type OVSLeaseStatus struct {
	StaticHosts []OVSStaticHostInfo `json:"static_hosts"`
	DHCPLeases  []OVSDHCPLeaseInfo  `json:"dhcp_leases"`
	Conflicts   []OVSLeaseConflict  `json:"conflicts"`
}

type OVSStaticHostInfo struct {
	VMName string `json:"vm_name"`
	MAC    string `json:"mac"`
	IP     string `json:"ip"`
}

type OVSDHCPLeaseInfo struct {
	ExpiryTime string `json:"expiry_time"`
	MAC        string `json:"mac"`
	IP         string `json:"ip"`
	Hostname   string `json:"hostname"`
	ClientID   string `json:"client_id"`
}

type OVSLeaseConflict struct {
	Type         string `json:"type"`
	IP           string `json:"ip"`
	MAC          string `json:"mac"`
	StaticVMName string `json:"static_vm_name"`
	LeaseHost    string `json:"lease_host"`
	Message      string `json:"message"`
}

type OVSCheckResult struct {
	Status            *OVSStatus      `json:"status"`
	Ports             *OVSPortList    `json:"ports"`
	Leases            *OVSLeaseStatus `json:"leases"`
	Healthy           bool            `json:"healthy"`
	RepairSuggestions []string        `json:"repair_suggestions"`
}

type VMNetworkRuntimeStatus struct {
	VMName     string                 `json:"vm_name"`
	State      string                 `json:"state"`
	Bridge     string                 `json:"bridge"`
	Interfaces []VMNetworkInterface   `json:"interfaces"`
	Bandwidth  OVSBandwidthReadStatus `json:"bandwidth"`
	Issues     []string               `json:"issues"`
}

type VMNetworkInterface struct {
	InterfaceType   string   `json:"interface_type"`
	Target          string   `json:"target"`
	SourceBridge    string   `json:"source_bridge"`
	SourceNetwork   string   `json:"source_network"`
	Model           string   `json:"model"`
	MAC             string   `json:"mac"`
	VirtualPortType string   `json:"virtualport_type"`
	OFPort          string   `json:"ofport"`
	IP              string   `json:"ip"`
	IPSource        string   `json:"ip_source"`
	Issues          []string `json:"issues"`
}

type OVSBandwidthReadStatus struct {
	Enabled         bool   `json:"enabled"`
	Cookie          string `json:"cookie"`
	FlowExists      bool   `json:"flow_exists"`
	DownQoS         bool   `json:"down_qos"`
	BridgeQoS       bool   `json:"bridge_qos"`
	TCRoot          bool   `json:"tc_root"`
	TCIngress       bool   `json:"tc_ingress"`
	TCUploadPolice  bool   `json:"tc_upload_police"`
	DownQueue       bool   `json:"down_queue"`
	UpQueue         bool   `json:"up_queue"`
	DownQueueID     uint32 `json:"down_queue_id"`
	UpQueueID       uint32 `json:"up_queue_id"`
	InboundAvgMbps  int    `json:"inbound_avg_mbps"`
	OutboundAvgMbps int    `json:"outbound_avg_mbps"`
	CheckedPort     string `json:"checked_port"`
	LastFlowError   string `json:"last_flow_error,omitempty"`
}

type ovsRuntimeInterface struct {
	Name   string
	Type   string
	Source string
	Model  string
	MAC    string
}

type ovsInterfaceXML struct {
	Type string `xml:"type,attr"`
	MAC  struct {
		Address string `xml:"address,attr"`
	} `xml:"mac"`
	Source struct {
		Bridge  string `xml:"bridge,attr"`
		Network string `xml:"network,attr"`
	} `xml:"source"`
	Target struct {
		Dev string `xml:"dev,attr"`
	} `xml:"target"`
	Model struct {
		Type string `xml:"type,attr"`
	} `xml:"model"`
	VirtualPort struct {
		Type string `xml:"type,attr"`
	} `xml:"virtualport"`
}

type ovsDomainXML struct {
	Devices struct {
		Interfaces []ovsInterfaceXML `xml:"interface"`
	} `xml:"devices"`
}

func GetOVSStatus() (*OVSStatus, error) {
	bridge := ovsBridgeName()
	subnet := ovsSubnetCIDR()
	uplink := ovsUplink()
	status := &OVSStatus{
		Bridge:     bridge,
		GatewayIP:  ovsGatewayIP(),
		SubnetCIDR: subnet,
		Uplink:     uplink,
		NATRule: OVSRuleStatus{
			Name:    "NAT MASQUERADE",
			Command: fmt.Sprintf("iptables -t nat -C POSTROUTING -s %s -o %s -j MASQUERADE", subnet, uplink),
		},
		ForwardOutRule: OVSRuleStatus{
			Name:    "OVS 出站转发",
			Command: fmt.Sprintf("iptables -C FORWARD -i %s -o %s -j ACCEPT", bridge, uplink),
		},
		ForwardReturnRule: OVSRuleStatus{
			Name:    "OVS 回程转发",
			Command: fmt.Sprintf("iptables -C FORWARD -i %s -o %s -m conntrack --ctstate RELATED,ESTABLISHED -j ACCEPT", uplink, bridge),
		},
	}

	bridgeResult := utils.ExecCommand("ovs-vsctl", "br-exists", bridge)
	status.BridgeExists = bridgeResult.Error == nil
	if bridgeResult.Error != nil {
		status.addIssue("OVS 网桥不存在或 ovs-vsctl 不可用")
		status.addCommandError("ovs-vsctl br-exists "+bridge, bridgeResult)
	}

	addrResult := utils.ExecCommand("ip", "-4", "addr", "show", "dev", bridge)
	status.BridgeHasGateway = addrResult.Error == nil && strings.Contains(addrResult.Stdout, status.GatewayIP+"/24")
	if !status.BridgeHasGateway {
		status.addIssue("OVS 网桥未配置预期网关 IP")
		if addrResult.Error != nil {
			status.addCommandError("ip -4 addr show dev "+bridge, addrResult)
		}
	}

	status.OpenVSwitchService = readSystemdServiceStatus("openvswitch-switch")
	status.DNSMasqService = readSystemdServiceStatus(ovsDNSMasqUnit)
	if !status.OpenVSwitchService.Active {
		status.addIssue("openvswitch-switch 服务未运行")
	}
	if !status.DNSMasqService.Active {
		status.addIssue("OVS dnsmasq 服务未运行")
	}

	ipForwardResult := utils.ExecCommand("sysctl", "-n", "net.ipv4.ip_forward")
	status.IPForwardEnabled = ipForwardResult.Error == nil && strings.TrimSpace(ipForwardResult.Stdout) == "1"
	if !status.IPForwardEnabled {
		status.addIssue("net.ipv4.ip_forward 未开启")
		if ipForwardResult.Error != nil {
			status.addCommandError("sysctl -n net.ipv4.ip_forward", ipForwardResult)
		}
	}

	if uplink == "" {
		status.addIssue("未检测到 OVS NAT 出口网卡")
	} else {
		status.NATRule.Exists = iptablesRuleExists("-t", "nat", "-C", "POSTROUTING", "-s", subnet, "-o", uplink, "-j", "MASQUERADE")
		status.ForwardOutRule.Exists = iptablesRuleExists("-C", "FORWARD", "-i", bridge, "-o", uplink, "-j", "ACCEPT")
		status.ForwardReturnRule.Exists = iptablesRuleExists("-C", "FORWARD", "-i", uplink, "-o", bridge, "-m", "conntrack", "--ctstate", "RELATED,ESTABLISHED", "-j", "ACCEPT")
		if !status.NATRule.Exists {
			status.addIssue("缺少 OVS NAT MASQUERADE 规则")
		}
		if !status.ForwardOutRule.Exists {
			status.addIssue("缺少 OVS 出站 FORWARD 放行规则")
		}
		if !status.ForwardReturnRule.Exists {
			status.addIssue("缺少 OVS 回程 FORWARD 放行规则")
		}
	}

	status.RepairSuggestions = buildOVSRepairSuggestions(status)
	status.Healthy = len(status.Issues) == 0
	return status, nil
}

func GetOVSPorts() (*OVSPortList, error) {
	bridge := ovsBridgeName()
	ofctlResult := utils.ExecCommand("ovs-ofctl", "-O", "OpenFlow13", "show", bridge)
	ports := parseOVSOfctlShow(ofctlResult.Stdout)
	if ofctlResult.Error != nil {
		return &OVSPortList{Bridge: bridge, Ports: []OVSPortStatus{}, Issues: []string{"读取 OVS 端口失败: " + ofctlResult.Stderr}}, nil
	}
	types := readOVSInterfaceTypes()
	vmInterfaces := collectAllVMRuntimeInterfaces()
	leases, _ := ListOVSDHCPLeases()
	staticHosts, _ := ListOVSStaticHosts()
	portList := correlateOVSPorts(ports, types, vmInterfaces, staticHosts, leases, bridge)
	return portList, nil
}

func GetOVSLeasesStatus() (*OVSLeaseStatus, error) {
	staticHosts, err := ListOVSStaticHosts()
	if err != nil {
		return nil, fmt.Errorf("读取 OVS 静态绑定失败: %w", err)
	}
	leases, err := ListOVSDHCPLeases()
	if err != nil {
		return nil, fmt.Errorf("读取 OVS DHCP 租约失败: %w", err)
	}
	return buildOVSLeaseStatus(staticHosts, leases), nil
}

func CheckOVSNetwork() (*OVSCheckResult, error) {
	status, err := GetOVSStatus()
	if err != nil {
		return nil, err
	}
	ports, err := GetOVSPorts()
	if err != nil {
		return nil, err
	}
	leases, err := GetOVSLeasesStatus()
	if err != nil {
		return nil, err
	}
	suggestions := append([]string{}, status.RepairSuggestions...)
	if len(ports.Issues) > 0 {
		suggestions = append(suggestions, "检查异常 OVS 端口，必要时确认 VM 网卡是否仍在运行")
	}
	if len(leases.Conflicts) > 0 {
		suggestions = append(suggestions, "处理 DHCP 租约与静态绑定冲突后重载 dnsmasq")
	}
	return &OVSCheckResult{
		Status:            status,
		Ports:             ports,
		Leases:            leases,
		Healthy:           status.Healthy && len(ports.Issues) == 0 && len(leases.Conflicts) == 0,
		RepairSuggestions: uniqueStrings(suggestions),
	}, nil
}

func RepairOVSNetwork(ctx context.Context, progress func(int, string)) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}
	if progress != nil {
		progress(10, "正在检查 OVS 网络配置...")
	}
	if err := EnsureOVSNetworkReady(); err != nil {
		return "", err
	}
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}
	if progress != nil {
		progress(80, "正在刷新 OVS 诊断状态...")
	}
	check, err := CheckOVSNetwork()
	if err != nil {
		return "", err
	}
	if progress != nil {
		progress(100, "OVS 网络修复任务已完成")
	}
	data, _ := json.Marshal(check)
	return string(data), nil
}

func GetVMNetworkRuntimeStatus(vmName string) (*VMNetworkRuntimeStatus, error) {
	vmName = strings.TrimSpace(vmName)
	if vmName == "" {
		return nil, fmt.Errorf("虚拟机名称不能为空")
	}
	status := &VMNetworkRuntimeStatus{
		VMName: vmName,
		Bridge: ovsBridgeName(),
	}
	stateResult := utils.ExecCommand("virsh", "domstate", vmName)
	if stateResult.Error == nil {
		status.State = strings.TrimSpace(stateResult.Stdout)
	} else {
		status.Issues = append(status.Issues, "读取 VM 状态失败: "+stateResult.Stderr)
	}
	running := strings.TrimSpace(status.State) == "running"

	xmlIfaces := readVMNetworkXML(vmName)
	runtimeIfaces := parseVirshDomiflistOutput(utils.ExecCommand("virsh", "domiflist", vmName).Stdout)
	runtimeByMAC := make(map[string]ovsRuntimeInterface)
	for _, item := range runtimeIfaces {
		runtimeByMAC[normalizeMAC(item.MAC)] = item
	}

	for _, iface := range xmlIfaces {
		item := VMNetworkInterface{
			InterfaceType:   iface.Type,
			Target:          iface.Target.Dev,
			SourceBridge:    iface.Source.Bridge,
			SourceNetwork:   iface.Source.Network,
			Model:           iface.Model.Type,
			MAC:             normalizeMAC(iface.MAC.Address),
			VirtualPortType: iface.VirtualPort.Type,
		}
		if runtime, ok := runtimeByMAC[item.MAC]; ok {
			if item.Target == "" || item.Target == "-" {
				item.Target = runtime.Name
			}
			if item.Model == "" {
				item.Model = runtime.Model
			}
			if item.SourceBridge == "" && runtime.Type == "bridge" {
				item.SourceBridge = runtime.Source
			}
		}
		if item.Target != "" && item.Target != "-" {
			item.OFPort = getOVSInterfaceOfPort(item.Target)
		}
		item.IP, item.IPSource = resolveVMIPByMAC(vmName, item.MAC, running)
		if item.SourceBridge == status.Bridge && item.VirtualPortType == "" {
			item.Issues = append(item.Issues, "网卡接入 OVS 网桥但缺少 openvswitch virtualport")
		}
		if item.SourceBridge == status.Bridge && (item.OFPort == "" || item.OFPort == "-1") && running {
			item.Issues = append(item.Issues, "运行态 OVS ofport 异常")
		}
		if item.SourceBridge == status.Bridge && item.IP == "" {
			item.Issues = append(item.Issues, "未找到 OVS 内网 IP")
		}
		status.Interfaces = append(status.Interfaces, item)
		status.Issues = append(status.Issues, item.Issues...)
	}

	if len(status.Interfaces) == 0 {
		status.Issues = append(status.Issues, "未读取到 VM 网卡配置")
	}
	status.Bandwidth = readOVSBandwidthStatus(vmName, status.Interfaces)
	return status, nil
}

func readSystemdServiceStatus(name string) OVSServiceStatus {
	result := utils.ExecCommand("systemctl", "is-active", name)
	state := strings.TrimSpace(result.Stdout)
	if state == "" {
		state = strings.TrimSpace(result.Stderr)
	}
	return OVSServiceStatus{
		Name:   name,
		Active: result.Error == nil && state == "active",
		State:  state,
		Error:  strings.TrimSpace(result.Stderr),
	}
}

func iptablesRuleExists(args ...string) bool {
	return utils.ExecCommand("iptables", args...).Error == nil
}

func (s *OVSStatus) addIssue(issue string) {
	s.Issues = append(s.Issues, issue)
}

func (s *OVSStatus) addCommandError(command string, result *utils.CmdResult) {
	if result == nil || result.Error == nil {
		return
	}
	message := strings.TrimSpace(result.Stderr)
	if message == "" {
		message = result.Error.Error()
	}
	s.RawCommandErrors = append(s.RawCommandErrors, OVSCommandFailure{Command: command, Error: message})
}

func buildOVSRepairSuggestions(status *OVSStatus) []string {
	var suggestions []string
	if !status.BridgeExists || !status.BridgeHasGateway {
		suggestions = append(suggestions, "重新确保 OVS 网桥和网关地址")
	}
	if !status.OpenVSwitchService.Active {
		suggestions = append(suggestions, "启动 openvswitch-switch 服务")
	}
	if !status.DNSMasqService.Active {
		suggestions = append(suggestions, "重写并启动 OVS dnsmasq 服务")
	}
	if !status.IPForwardEnabled {
		suggestions = append(suggestions, "开启 IPv4 转发")
	}
	if !status.NATRule.Exists || !status.ForwardOutRule.Exists || !status.ForwardReturnRule.Exists {
		suggestions = append(suggestions, "补齐 NAT 和 FORWARD 规则")
	}
	return uniqueStrings(suggestions)
}

func parseOVSOfctlShow(text string) []OVSPortStatus {
	re := regexp.MustCompile(`^\s*(LOCAL|\d+)\(([^)]+)\):`)
	var ports []OVSPortStatus
	for _, line := range strings.Split(text, "\n") {
		matches := re.FindStringSubmatch(line)
		if len(matches) != 3 {
			continue
		}
		ports = append(ports, OVSPortStatus{
			OFPort: strings.TrimSpace(matches[1]),
			Name:   strings.TrimSpace(matches[2]),
		})
	}
	return ports
}

func readOVSInterfaceTypes() map[string]string {
	result := utils.ExecCommand("ovs-vsctl", "--format=csv", "--data=bare", "--no-heading", "--columns=name,type", "list", "Interface")
	return parseOVSInterfaceTypeCSV(result.Stdout)
}

func parseOVSInterfaceTypeCSV(text string) map[string]string {
	types := make(map[string]string)
	reader := csv.NewReader(strings.NewReader(text))
	reader.TrimLeadingSpace = true
	records, err := reader.ReadAll()
	if err != nil {
		return types
	}
	for _, record := range records {
		if len(record) < 1 {
			continue
		}
		name := strings.TrimSpace(record[0])
		if name == "" {
			continue
		}
		if len(record) >= 2 && strings.TrimSpace(record[1]) != "" {
			types[name] = strings.TrimSpace(record[1])
		} else {
			types[name] = "system"
		}
	}
	return types
}

func collectAllVMRuntimeInterfaces() map[string]ovsRuntimeInterface {
	interfaces := make(map[string]ovsRuntimeInterface)
	for _, vmName := range listAllVMNames() {
		result := utils.ExecCommand("virsh", "domiflist", vmName)
		for _, iface := range parseVirshDomiflistOutput(result.Stdout) {
			if iface.MAC == "" {
				continue
			}
			iface.Name = strings.TrimSpace(iface.Name)
			interfaces[iface.Name] = iface
			interfaces[normalizeMAC(iface.MAC)] = ovsRuntimeInterface{
				Name:   vmName,
				Type:   iface.Type,
				Source: iface.Source,
				Model:  iface.Model,
				MAC:    iface.MAC,
			}
		}
	}
	return interfaces
}

func parseVirshDomiflistOutput(text string) []ovsRuntimeInterface {
	var result []ovsRuntimeInterface
	macRe := regexp.MustCompile(`(?i)^([0-9a-f]{2}:){5}[0-9a-f]{2}$`)
	for _, line := range strings.Split(text, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 5 || fields[0] == "Interface" {
			continue
		}
		mac := normalizeMAC(fields[4])
		if !macRe.MatchString(mac) {
			continue
		}
		result = append(result, ovsRuntimeInterface{
			Name:   fields[0],
			Type:   fields[1],
			Source: fields[2],
			Model:  fields[3],
			MAC:    mac,
		})
	}
	return result
}

func correlateOVSPorts(ports []OVSPortStatus, types map[string]string, vmInterfaces map[string]ovsRuntimeInterface, staticHosts []OVSStaticHost, leases []OVSDHCPLease, bridge string) *OVSPortList {
	staticByMAC := make(map[string]OVSStaticHost)
	leaseByMAC := make(map[string]OVSDHCPLease)
	for _, host := range staticHosts {
		staticByMAC[normalizeMAC(host.MAC)] = host
	}
	for _, lease := range leases {
		mac := normalizeMAC(lease.MAC)
		leaseByMAC[mac] = newerOVSDHCPLease(leaseByMAC[mac], lease)
	}

	var issues []string
	for i := range ports {
		port := &ports[i]
		port.Type = types[port.Name]
		if port.Type == "" {
			port.Type = "system"
		}
		if runtime, ok := vmInterfaces[port.Name]; ok {
			port.MAC = normalizeMAC(runtime.MAC)
			if vmRuntime, ok := vmInterfaces[port.MAC]; ok {
				port.VMName = vmRuntime.Name
			}
		}
		if port.MAC != "" {
			if host, ok := staticByMAC[port.MAC]; ok {
				port.IP = host.IP
				port.IPSource = "static"
				if port.VMName == "" {
					port.VMName = host.VMName
				}
			} else if lease, ok := leaseByMAC[port.MAC]; ok {
				port.IP = lease.IP
				port.IPSource = "dhcp"
			}
		}
		isVMPort := port.OFPort != "LOCAL" && port.Name != bridge && (strings.HasPrefix(port.Name, "vnet") || strings.HasPrefix(port.Name, "tap") || port.MAC != "")
		if port.OFPort == "-1" {
			port.Issues = append(port.Issues, "ofport 为 -1")
		}
		if isVMPort && port.VMName == "" {
			port.Issues = append(port.Issues, "未关联 VM")
		}
		if isVMPort && port.IP == "" {
			port.Issues = append(port.Issues, "未找到 IP")
		}
		for _, issue := range port.Issues {
			issues = append(issues, fmt.Sprintf("%s: %s", port.Name, issue))
		}
	}
	sort.Slice(ports, func(i, j int) bool {
		return ports[i].Name < ports[j].Name
	})
	return &OVSPortList{Bridge: bridge, Ports: ports, Issues: issues}
}

func buildOVSLeaseStatus(staticHosts []OVSStaticHost, leases []OVSDHCPLease) *OVSLeaseStatus {
	status := &OVSLeaseStatus{}
	for _, host := range staticHosts {
		status.StaticHosts = append(status.StaticHosts, OVSStaticHostInfo{VMName: host.VMName, MAC: normalizeMAC(host.MAC), IP: host.IP})
	}
	for _, lease := range leases {
		status.DHCPLeases = append(status.DHCPLeases, OVSDHCPLeaseInfo{
			ExpiryTime: lease.ExpiryTime,
			MAC:        normalizeMAC(lease.MAC),
			IP:         lease.IP,
			Hostname:   lease.Hostname,
			ClientID:   lease.ClientID,
		})
	}
	status.Conflicts = detectOVSLeaseConflicts(staticHosts, leases)
	return status
}

func detectOVSLeaseConflicts(staticHosts []OVSStaticHost, leases []OVSDHCPLease) []OVSLeaseConflict {
	var conflicts []OVSLeaseConflict
	for _, host := range staticHosts {
		hostMAC := normalizeMAC(host.MAC)
		for _, lease := range leases {
			leaseMAC := normalizeMAC(lease.MAC)
			if host.IP == lease.IP && hostMAC != leaseMAC {
				conflicts = append(conflicts, OVSLeaseConflict{
					Type:         "ip_conflict",
					IP:           host.IP,
					MAC:          leaseMAC,
					StaticVMName: host.VMName,
					LeaseHost:    lease.Hostname,
					Message:      fmt.Sprintf("静态 IP %s 已绑定到 %s，但 DHCP 租约被 %s 使用", host.IP, hostMAC, leaseMAC),
				})
			}
			if hostMAC == leaseMAC && host.IP != lease.IP {
				conflicts = append(conflicts, OVSLeaseConflict{
					Type:         "mac_conflict",
					IP:           lease.IP,
					MAC:          hostMAC,
					StaticVMName: host.VMName,
					LeaseHost:    lease.Hostname,
					Message:      fmt.Sprintf("MAC %s 静态绑定为 %s，但 DHCP 租约为 %s", hostMAC, host.IP, lease.IP),
				})
			}
		}
	}
	return conflicts
}

func readVMNetworkXML(vmName string) []ovsInterfaceXML {
	result := utils.ExecCommand("virsh", "dumpxml", vmName)
	if result.Error != nil {
		return []ovsInterfaceXML{}
	}
	var dom ovsDomainXML
	if err := xml.Unmarshal([]byte(result.Stdout), &dom); err != nil {
		return []ovsInterfaceXML{}
	}
	return dom.Devices.Interfaces
}

func resolveVMIPByMAC(vmName, mac string, running bool) (string, string) {
	mac = normalizeMAC(mac)
	if running {
		if ip, ok := firstVirshDomifaddrIP(vmName, "agent"); ok {
			return ip, "guest_agent"
		}
		if ip := GetVPCLeaseIPForVM(vmName); ip != "" {
			return ip, "ovs_dhcp"
		}
		if ip := GetOVSStaticIPByMAC(mac); ip != "" {
			return ip, "static"
		}
		if ip := GetOVSLeaseIPByMAC(mac); ip != "" {
			return ip, "ovs_dhcp"
		}
		if ip, ok := firstVirshDomifaddrIP(vmName, "arp"); ok {
			return ip, "arp"
		}
		if ip, ok := firstVirshDomifaddrIP(vmName, "lease"); ok {
			return ip, "libvirt_lease"
		}
	}
	if ip := GetOVSStaticIPByMAC(mac); ip != "" {
		return ip, "static"
	}
	return "", ""
}

func firstVirshDomifaddrIP(vmName, source string) (string, bool) {
	result := utils.ExecCommand("virsh", "domifaddr", vmName, "--source", source)
	if result.Error != nil {
		return "", false
	}
	ipRe := regexp.MustCompile(`(\d+\.\d+\.\d+\.\d+)`)
	matches := ipRe.FindAllStringSubmatch(result.Stdout, -1)
	for _, match := range matches {
		ip := match[1]
		if ip != "127.0.0.1" && net.ParseIP(ip) != nil {
			return ip, true
		}
	}
	return "", false
}

func readOVSBandwidthStatus(vmName string, interfaces []VMNetworkInterface) OVSBandwidthReadStatus {
	status := OVSBandwidthReadStatus{
		Cookie:      ovsBandwidthCookie(vmName),
		DownQueueID: ovsBandwidthQueueID(vmName, "down"),
		UpQueueID:   ovsBandwidthQueueID(vmName, "up"),
	}
	if detail, err := GetVMBandwidth(vmName); err == nil && detail != nil {
		status.InboundAvgMbps = detail.InboundAvg
		status.OutboundAvgMbps = detail.OutboundAvg
	}
	for _, iface := range interfaces {
		if iface.Target != "" && iface.Target != "-" && iface.SourceBridge == ovsBridgeName() {
			status.CheckedPort = iface.Target
			break
		}
	}
	if status.CheckedPort != "" {
		qos := strings.TrimSpace(utils.ExecCommand("ovs-vsctl", "get", "Port", status.CheckedPort, "qos").Stdout)
		status.DownQoS = qos != "" && qos != "[]"
		qdisc := utils.ExecCommand("tc", "qdisc", "show", "dev", status.CheckedPort)
		if qdisc.Error == nil {
			status.TCRoot = strings.Contains(qdisc.Stdout, "htb")
			status.TCIngress = strings.Contains(qdisc.Stdout, "ingress")
		}
		filter := utils.ExecCommand("tc", "filter", "show", "dev", status.CheckedPort, "ingress")
		if filter.Error == nil {
			status.TCUploadPolice = strings.Contains(filter.Stdout, "police")
		}
	}
	bridgeQoS := strings.TrimSpace(utils.ExecCommand("ovs-vsctl", "get", "Port", ovsBridgeName(), "qos").Stdout)
	status.BridgeQoS = bridgeQoS != "" && bridgeQoS != "[]"
	status.DownQueue = len(findOVSUUIDs("Queue", vmName, "down")) > 0
	status.UpQueue = len(findOVSUUIDs("Queue", vmName, "up")) > 0
	flowResult := utils.ExecCommand("ovs-ofctl", "-O", "OpenFlow13", "dump-flows", ovsBridgeName())
	if flowResult.Error == nil {
		status.FlowExists = hasOVSBandwidthFlow(flowResult.Stdout, status.Cookie)
	} else {
		status.LastFlowError = strings.TrimSpace(flowResult.Stderr)
	}
	status.Enabled = status.FlowExists || status.DownQoS || status.BridgeQoS || status.DownQueue || status.UpQueue || status.TCRoot || status.TCIngress || status.TCUploadPolice
	return status
}

func hasOVSBandwidthFlow(text, cookie string) bool {
	cookie = strings.ToLower(strings.TrimSpace(cookie))
	if cookie == "" {
		return false
	}
	for _, line := range strings.Split(strings.ToLower(text), "\n") {
		if strings.Contains(line, "cookie="+cookie) || strings.Contains(line, "cookie="+strings.TrimPrefix(cookie, "0x")) {
			return true
		}
	}
	return false
}

func normalizeMAC(mac string) string {
	return strings.ToLower(strings.TrimSpace(mac))
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	return result
}
