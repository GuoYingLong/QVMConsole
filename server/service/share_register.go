package service

import sharepkg "kvm_console/service/share"

// share_register.go — 将 service 根包函数注入到 share 子包的 Hook 变量中，
// 供 share 子包通过 Hook 间接调用根包函数，避免循环 import。
// 当前 share 子包不依赖 service 根包函数，保留空 init 以遵循架构模式。

func init() {
	_ = sharepkg.ShareInfo{}
}
