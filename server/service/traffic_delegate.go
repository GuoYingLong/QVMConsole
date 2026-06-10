package service

// Traffic function delegates - forward to service/traffic subpackage
// Maintains backward compatibility for callers using service.XXX()

import traffpkg "kvm_console/service/traffic"

// ── Exported delegates ──

func AggregateUserDailyTraffic(username string) (downBytes, upBytes int64) {
	return traffpkg.AggregateUserDailyTraffic(username)
}

func GetUserTrafficUsage(username string) *TrafficUsageInfo {
	return traffpkg.GetUserTrafficUsage(username)
}

func CheckAndApplyTrafficLimit(username string) {
	traffpkg.CheckAndApplyTrafficLimit(username)
}

func ResetUserTrafficQuota(username string) error {
	return traffpkg.ResetUserTrafficQuota(username)
}

func ResetAllDailyTraffic() {
	traffpkg.ResetAllDailyTraffic()
}

func CheckAllUsersTrafficQuota() {
	traffpkg.CheckAllUsersTrafficQuota()
}

func CheckTrafficAfterQuotaUpdate(username string) {
	traffpkg.CheckTrafficAfterQuotaUpdate(username)
}

func IsUserTrafficLimited(username string) (downLimited, upLimited bool) {
	return traffpkg.IsUserTrafficLimited(username)
}

func CheckUserTrafficQuotaForStart(username string) error {
	return traffpkg.CheckUserTrafficQuotaForStart(username)
}

func StartTrafficQuotaChecker() {
	traffpkg.StartTrafficQuotaChecker()
}

// ── Unexported delegates（供 service 根包其他 register 文件使用） ──

// formatTrafficBytes delegates to traffic.FormatTrafficBytes
func formatTrafficBytes(bytes int64) string {
	return traffpkg.FormatTrafficBytes(bytes)
}
