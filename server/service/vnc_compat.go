package service

import vncpkg "kvm_console/service/vnc"

// ── Type aliases（向后兼容，让 service 根包和外部调用方可直接使用类型名） ──

type VncInfo = vncpkg.VncInfo
type VncConnInfo = vncpkg.VncConnInfo
