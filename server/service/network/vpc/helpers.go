package vpc

import (
	"fmt"
	"net"
	"net/netip"
	"regexp"
	"strings"
	"time"

	"kvm_console/logger"
	"kvm_console/model"
)

func CurrentTrafficMonth() string {
	return time.Now().Format("2006-01")
}

func normalizeVPCName(value string) string {
	value = strings.TrimSpace(value)
	value = regexp.MustCompile(`\s+`).ReplaceAllString(value, "-")
	return value
}

func normalizeVPCSwitchBandwidthRequest(req *VPCSwitchRequest) {
	if req == nil {
		return
	}
	if req.BandwidthDownMbps <= 0 && req.BandwidthUpMbps <= 0 && req.BandwidthMbps > 0 {
		req.BandwidthDownMbps = req.BandwidthMbps
		req.BandwidthUpMbps = req.BandwidthMbps
	}
	if req.BandwidthDownMbps < 0 {
		req.BandwidthDownMbps = 0
	}
	if req.BandwidthUpMbps < 0 {
		req.BandwidthUpMbps = 0
	}
	if req.BandwidthMbps <= 0 && req.BandwidthDownMbps == req.BandwidthUpMbps {
		req.BandwidthMbps = req.BandwidthDownMbps
	}
}

func normalizeVPCSwitchBandwidthForResponse(sw *model.VPCSwitch) {
	if sw == nil {
		return
	}
	if sw.BandwidthDownMbps <= 0 && sw.BandwidthUpMbps <= 0 && sw.BandwidthMbps > 0 {
		sw.BandwidthDownMbps = sw.BandwidthMbps
		sw.BandwidthUpMbps = sw.BandwidthMbps
	}
}

func fillVPCSwitchUsageForResponse(sw *model.VPCSwitch) {
	if sw == nil {
		return
	}
	normalizeVPCSwitchBandwidthForResponse(sw)
	down, up := AggregateSwitchMonthlyTraffic(sw.ID)
	sw.UsedTrafficDown = down
	sw.UsedTrafficUp = up
	sw.UsedTrafficDownGB = HookFormatTrafficBytes(down)
	sw.UsedTrafficUpGB = HookFormatTrafficBytes(up)
	sw.IsLimitedDown, sw.IsLimitedUp = IsVPCSwitchTrafficLimited(sw.ID)
	sw.EffectiveBandwidthDownMbps, sw.EffectiveBandwidthUpMbps = effectiveVPCSwitchBandwidth(*sw)
}

func resolveVPCUsername(operator, role, requested string) (string, error) {
	requested = strings.TrimSpace(requested)
	if role == "admin" && requested != "" {
		return requested, nil
	}
	if strings.TrimSpace(operator) == "" {
		return "", fmt.Errorf("无法识别当前用户")
	}
	return operator, nil
}

// EnsureDefaultSecurityGroup 确保用户存在默认安全组。
func EnsureDefaultSecurityGroup(username string) (*model.VPCSecurityGroup, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return nil, fmt.Errorf("用户名不能为空")
	}
	var group model.VPCSecurityGroup
	if err := model.DB.Where("username = ? AND is_default = ?", username, true).First(&group).Error; err == nil {
		return &group, nil
	}
	group = model.VPCSecurityGroup{
		Username:  username,
		Name:      "默认安全组",
		IsDefault: true,
		Remark:    "系统自动创建，默认拒绝入站、允许出站",
	}
	if err := model.DB.Create(&group).Error; err != nil {
		return nil, fmt.Errorf("创建默认安全组失败: %w", err)
	}
	return &group, nil
}

func EnsureDefaultVPCSwitch(username string) (*model.VPCSwitch, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return nil, fmt.Errorf("用户名不能为空")
	}
	var user model.User
	if err := model.DB.Where("username = ?", username).First(&user).Error; err != nil {
		return nil, fmt.Errorf("用户不存在")
	}
	if user.Role != "user" {
		return nil, nil
	}
	var sw model.VPCSwitch
	if err := model.DB.Where("username = ?", username).Order("id ASC").First(&sw).Error; err == nil {
		return &sw, nil
	}
	req := defaultVPCSwitchRequestForUser(user)
	created, err := CreateVPCSwitch("system", "admin", req)
	if err != nil && created != nil {
		logger.App.Warn("默认交换机已创建但运行态应用失败", "switch", created.Name, "error", err)
		return created, nil
	}
	return created, err
}

func defaultVPCSwitchRequestForUser(user model.User) VPCSwitchRequest {
	return VPCSwitchRequest{
		Username:          user.Username,
		Name:              DefaultVPCSwitchName,
		TrafficDownGB:     defaultSwitchTrafficQuota(user.MaxTrafficDown),
		TrafficUpGB:       defaultSwitchTrafficQuota(user.MaxTrafficUp),
		BandwidthDownMbps: defaultSwitchBandwidthQuota(user.MaxBandwidthDown),
		BandwidthUpMbps:   defaultSwitchBandwidthQuota(user.MaxBandwidthUp),
	}
}

func defaultSwitchTrafficQuota(max float64) float64 {
	if max > 0 {
		return max
	}
	return 0
}

func defaultSwitchBandwidthQuota(max float64) int {
	if max <= 0 {
		return 0
	}
	value := int(max)
	if value <= 0 {
		return 1
	}
	return value
}

func EnsureAllActiveUsersDefaultSecurityGroup() {
	var users []model.User
	model.DB.Where("role = ? AND status = ?", "user", UserStatusActive).Find(&users)
	for _, user := range users {
		if _, err := EnsureDefaultSecurityGroup(user.Username); err != nil {
			logger.App.Warn("创建默认安全组失败", "user", user.Username, "error", err)
		}
	}
}

func GetVPCQuota(username string) (*VPCQuotaInfo, error) {
	var user model.User
	if err := model.DB.Where("username = ?", username).First(&user).Error; err != nil {
		return nil, fmt.Errorf("用户不存在")
	}
	var switches []model.VPCSwitch
	model.DB.Where("username = ?", username).Find(&switches)
	info := &VPCQuotaInfo{
		Username:         username,
		MaxTrafficDown:   user.MaxTrafficDown,
		MaxTrafficUp:     user.MaxTrafficUp,
		MaxBandwidthDown: user.MaxBandwidthDown,
		MaxBandwidthUp:   user.MaxBandwidthUp,
	}
	for _, sw := range switches {
		info.AllocatedTrafficDown += sw.TrafficDownGB
		info.AllocatedTrafficUp += sw.TrafficUpGB
		info.AllocatedBandwidthDown += float64(sw.BandwidthDownMbps)
		info.AllocatedBandwidthUp += float64(sw.BandwidthUpMbps)
	}
	if user.MaxTrafficDown > 0 {
		info.RemainingTrafficDown = user.MaxTrafficDown - info.AllocatedTrafficDown
	}
	if user.MaxTrafficUp > 0 {
		info.RemainingTrafficUp = user.MaxTrafficUp - info.AllocatedTrafficUp
	}
	if user.MaxBandwidthDown > 0 {
		info.RemainingBandwidthDown = user.MaxBandwidthDown - info.AllocatedBandwidthDown
	}
	if user.MaxBandwidthUp > 0 {
		info.RemainingBandwidthUp = user.MaxBandwidthUp - info.AllocatedBandwidthUp
	}
	if info.RemainingTrafficDown < 0 {
		info.RemainingTrafficDown = 0
	}
	if info.RemainingTrafficUp < 0 {
		info.RemainingTrafficUp = 0
	}
	if info.RemainingBandwidthDown < 0 {
		info.RemainingBandwidthDown = 0
	}
	if info.RemainingBandwidthUp < 0 {
		info.RemainingBandwidthUp = 0
	}
	return info, nil
}

func normalizeCIDROrIP(value string) string {
	value = strings.TrimSpace(value)
	if addr, err := netip.ParseAddr(value); err == nil && addr.Is4() {
		return addr.String() + "/32"
	}
	return value
}

func IPInCIDR(ipText, cidrText string) bool {
	ip := net.ParseIP(strings.TrimSpace(ipText))
	_, network, err := net.ParseCIDR(strings.TrimSpace(cidrText))
	return ip != nil && err == nil && network.Contains(ip)
}

func IsVPCManagedIP(ipText string) bool {
	ipText = strings.TrimSpace(ipText)
	if ipText == "" || model.DB == nil {
		return false
	}
	var switches []model.VPCSwitch
	model.DB.Find(&switches)
	for _, sw := range switches {
		if IPInCIDR(ipText, sw.CIDR) {
			return true
		}
	}
	return false
}
