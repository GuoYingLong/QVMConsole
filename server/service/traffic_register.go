package service

import (
	traffpkg "kvm_console/service/traffic"
	lwpkg "kvm_console/service/lightweight"
	vpcpkg "kvm_console/service/network/vpc"
)

// traffic_register.go — 将 service 根包函数注入到 traffic 子包的 Hook 变量中，
// 供 traffic 子包通过 Hook 间接调用根包函数，避免循环 import。

func init() {
	// ── 向 vpc / lightweight 子包注册 traffic 工具函数 ──
	vpcpkg.HookFormatTrafficBytes = traffpkg.FormatTrafficBytes
	lwpkg.HookFormatTrafficBytes = traffpkg.FormatTrafficBytes
}
