package bridge

const (
	BridgeModeNAT    = "nat"
	BridgeModeDirect = "bridge"
	bridgeConfigDir  = "/etc/kvm-console/bridges"
)

type HostInterfaceInfo struct {
	Name          string   `json:"name"`
	MAC           string   `json:"mac"`
	State         string   `json:"state"`
	MTU           int      `json:"mtu"`
	Addresses     []string `json:"addresses"`
	DefaultRoute  bool     `json:"default_route"`
	OVSBridge     string   `json:"ovs_bridge"`
	OVSPort       bool     `json:"ovs_port"`
	Physical      bool     `json:"physical"`
	ManagedBridge string   `json:"managed_bridge"`
	Risk          string   `json:"risk,omitempty"`
}

type NetworkBridgeInfo struct {
	ID            uint   `json:"id"`
	Name          string `json:"name"`
	Mode          string `json:"mode"`
	UplinkIF      string `json:"uplink_if"`
	MigrateHostIP bool   `json:"migrate_host_ip"`
	IsDefault     bool   `json:"is_default"`
	Exists        bool   `json:"exists"`
	Active        bool   `json:"active"`
	SwitchCount   int64  `json:"switch_count"`
}

type NetworkBridgeRequest struct {
	Name          string `json:"name"`
	Mode          string `json:"mode"`
	UplinkIF      string `json:"uplink_if"`
	MigrateHostIP bool   `json:"migrate_host_ip"`
}

type ipAddrJSON struct {
	IfName    string `json:"ifname"`
	Address   string `json:"address"`
	OperState string `json:"operstate"`
	MTU       int    `json:"mtu"`
	AddrInfo  []struct {
		Local     string `json:"local"`
		PrefixLen int    `json:"prefixlen"`
		Family    string `json:"family"`
	} `json:"addr_info"`
}

type ipRouteJSON struct {
	Dst     string `json:"dst"`
	Dev     string `json:"dev"`
	Gateway string `json:"gateway"`
}
