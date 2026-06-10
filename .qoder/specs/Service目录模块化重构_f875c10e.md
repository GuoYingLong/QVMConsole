# Service 目录模块化重构执行计划

## Context

当前 `server/service/` 目录包含 128 个 Go 文件全部在同一 package 中，最大文件 vpc.go 达 3409 行，8 个文件超过 1500 行。单包臃肿导致职责混杂、修改影响面大、新人理解成本高。本次改造将其按业务边界拆分为子包，同时顺手优化 handler 层的 vm.go 和 user.go。

**关键约束：**
- 不改变任何函数签名和外部行为
- handler 层仅修改 import 路径
- 每阶段编译通过后才进入下一阶段
- Module path: `kvm_console`

---

## 阶段一：提取基础设施子包（零风险）

目标：提取无业务逻辑的共用层，为后续阶段建立基础。

### Task 1.1 — 创建 `service/libvirt_rpc/` 子包

**源文件：** `libvirt_rpc.go`(113行), `libvirt_rpc_domain.go`(868行)

**目标结构：**
```
service/libvirt_rpc/
├── connection.go      # GetLibvirt, IsLibvirtRPCAvailable, 连接管理
└── domain.go          # getDomainInfoRPC, getDomainXMLRPC, defineDomainXMLRPC 等
```

**操作：**
1. 创建子包 `package libvirt_rpc`
2. 将导出函数大写化（供 service 包调用）
3. 更新 service 包中所有调用处 import `kvm_console/server/service/libvirt_rpc`
4. 删除原文件

### Task 1.2 — 创建 `service/vm_xml/` 子包

**源文件提取自：** libvirt.go, clone.go, disk.go, vm_dynamic_memory.go, vm_boot_type.go

**目标结构：**
```
service/vm_xml/
├── parse_helpers.go    # parseInfoInt, parseMemStat, parseIfStat, parseBlkStat
├── domain_xml.go       # ParseDisksFromDomainXML, FillVmInfoFromXML
├── boot_type.go        # ParseVMBootTypeFromDomainXML, ApplyVMBootTypeToDomainXML
├── cpu_xml.go          # ParseVCPUCountFromDomainXML, ApplyCPUTopologyMode
├── memory_xml.go       # ApplyDynamicMemoryConfigToDomainXML, ApplyVirtioMemConfigToDomainXML
├── device_xml.go       # ParseVMVideoModel, ApplyVMVideoModel, ApplyVMGuestAgent
├── smbios_xml.go       # ApplySMBIOS1ConfigToDomainXML
├── rtc_xml.go          # ApplyRTCConfigToDomainXML
└── boot_devices.go     # parseBootDevices, BootDevice 类型
```

**操作：**
1. 提取所有 XML 解析/生成相关函数
2. 类型定义（如 BootDevice）随函数迁移
3. 更新所有调用方的 import

### Task 1.3 — 创建 `service/ip_resolver/` 子包

**源文件提取自：** libvirt.go (getVMIP, getFirstVMMAC, getVMMAC 等)

**目标结构：**
```
service/ip_resolver/
├── resolver.go    # GetVMIP（统一入口）
├── mac.go         # GetFirstVMMAC, GetVMMAC
├── arp.go         # ARP 表扫描、主动探测
├── vpc.go         # VPC 相关 IP 查找
└── neighbor.go    # ip neigh 查询
```

### Task 1.4 — 提取带宽类型与工具函数

**源文件：** bandwidth.go

**提取到：** `service/bandwidth/types.go`
- BandwidthInfo, BandwidthDetail 等结构体
- MbpsToKBps, KBpsToMbps 等转换函数
- buildBandwidthParams 等参数构建函数

**阶段一验证：** `go build ./server/...` 编译通过

---

## 阶段二：拆解核心上帝文件

### Task 2.1 — 拆解 libvirt.go（1892行 → vm/ 子包）

**目标结构：**
```
service/vm/
├── types.go            # VmInfo, VmDetail, VmStats, HostStats, VMListOptions
├── lifecycle.go        # StartVM, ShutdownVM, DestroyVM, RebootVM, ResetVM
├── list.go             # ListVMs, GetVM, GetVMIPInfo
├── config.go           # EditVMConfig, SetVMBootOrder, SetVMAutostart, SetVMNicModel
├── freeze.go           # GetVMFreeze, SetVMFreeze
├── remark.go           # GetVMRemark, SetVMRemark
├── owner.go            # FindVMOwner, GetUserVMList
├── runtime.go          # VMRuntimeInfo, 持续运行追踪
└── stats.go            # GetVMStats, GetHostStats（宿主机统计部分移至 host/）
```

### Task 2.2 — 拆解 clone.go（2171行 → clone/ 子包）

**目标结构：**
```
service/clone/
├── types.go            # CloneParams, BatchCloneParams, ReinstallParams
├── clone.go            # CloneVM 主逻辑
├── batch_clone.go      # BatchCloneVM
├── reinstall.go        # ReinstallVM
├── delete.go           # DeleteVM, DeleteVMWithDisks, ForceDeleteVM
├── init_linux.go       # initLinuxClone, prepareLinuxCloneFirstBootIdentity
├── init_windows.go     # cloneWindows, buildWindowsUnattendXML
├── init_fnos.go        # cloneFnOS, buildFnOS*Command
└── xml_pipeline.go     # XML 注入管线
```

### Task 2.3 — 拆解 snapshot.go（1773行 → snapshot/ 子包）

**目标结构：**
```
service/snapshot/
├── types.go        # SnapshotInfo 等结构体
├── core.go         # CreateSnapshot, DeleteSnapshot, RevertSnapshot, ListSnapshots
├── external.go     # 外部快照操作
├── chain.go        # 磁盘链合并
├── nvram.go        # UEFI NVRAM 处理
├── apparmor.go     # AppArmor 规则管理
├── quota.go        # 配额检查
└── cleanup.go      # 残留清理
```

### Task 2.4 — 拆解 network.go（1594行 → network/ 子包）

**目标结构：**
```
service/network/
├── port_forward/
│   ├── types.go
│   ├── rules.go
│   ├── persistence.go
│   └── availability.go
├── static_ip/
│   ├── ovs.go
│   ├── vpc.go
│   └── auto.go
└── diagnostics/
    └── diagnostics.go
```

### Task 2.5 — 拆解 disk.go（1392行 → storage/disk/ 子包）

**目标结构：**
```
service/storage/
├── disk/
│   ├── types.go
│   ├── crud.go
│   ├── cdrom.go
│   ├── iops.go
│   └── pcie.go
└── pool/
    └── pool.go        # storage_pool.go 现有逻辑
```

### Task 2.6 — 拆解 vpc.go（3409行 → network/vpc/ 子包）

**目标结构：**
```
service/network/vpc/
├── types.go            # VPC/VSwitch 类型定义
├── switch.go           # 虚拟交换机 CRUD
├── routing.go          # VPC 路由规则
├── dhcp.go             # DHCP 服务管理
└── ovs_bridge.go       # OVS 桥接操作
```

### Task 2.7 — 拆解 vm_migration.go（1824行 → vm/migration/ 子包）

**目标结构：**
```
service/vm/migration/
├── types.go            # 迁移状态、参数类型
├── migrate.go          # MigrateVM 主逻辑
├── dirty_rate.go       # 脏页率检测
├── state.go            # 迁移状态管理
└── lock.go             # 迁移锁机制
```

### Task 2.8 — 拆解 vm_import.go（1300行 → vm/import/ 子包）

**目标结构：**
```
service/vm/vmimport/
├── types.go            # 导入参数类型
├── import.go           # ImportVM 主逻辑
├── disk_convert.go     # 磁盘格式转换
└── detect.go           # 系统检测
```

### Task 2.9 — 拆解 vm_dynamic_memory.go（1142行）

**目标结构：**
```
service/vm/memory/
├── types.go            # 动态内存配置结构体
├── metadata.go         # readVMMemoryMetadata, writeVMMemoryMetadata
├── balloon.go          # Balloon 控制
├── virtiomem.go        # VirtioMem 控制
├── scheduler.go        # 内存调度器
└── config.go           # SetVMMemoryDynamicConfig, ApplyPendingVMMemoryConfig
```

**阶段二验证：** `go build ./server/...` 编译通过

---

## 阶段三：提取功能子包（低风险，增强内聚）

### Task 3.1 — bandwidth/ 子包完整化
将 bandwidth.go 剩余业务逻辑移入 `service/bandwidth/`：tc.go, ovs.go, vm.go, quota.go, global.go

### Task 3.2 — public_ip/ 子包
将 public_ip.go (962行) 拆分为 `service/public_ip/`：crud.go, binding.go, nat.go, rules.go

### Task 3.3 — firewall/ 子包
合并 firewall.go + host_firewall.go → `service/firewall/`：policy.go, host.go, ufw.go

### Task 3.4 — security/ 子包
归并 security_*.go 系列文件 → `service/security/`

### Task 3.5 — lightweight/ 子包
合并 lightweight_*.go → `service/lightweight/`：cloud.go, registration.go, quota.go

### Task 3.6 — user/ 子包
合并 user*.go → `service/user/`：core.go, quota.go, storage.go, ssh.go

### Task 3.7 — host/ 子包
合并 host_*.go + maintenance.go → `service/host/`：stats.go, node.go, ksm.go, zram.go, maintenance.go

### Task 3.8 — 其他小模块
- `service/scheduler/` ← scheduler_center.go
- `service/traffic/` ← traffic_quota.go
- `service/ovs/` ← ovs_network.go, ovs_diagnostics.go
- `service/rescue/` ← rescue.go
- `service/vnc/` ← vnc.go
- `service/share/` ← share.go

**阶段三验证：** `go build ./server/...` 编译通过

---

## 阶段四：消除重复逻辑 + handler 优化

### Task 4.1 — 统一 IP 获取
确保 network/bandwidth/public_ip/clone 均通过 `ip_resolver` 获取 IP，消除重复实现。

### Task 4.2 — 统一 XML 操作
确保所有 XML 修改通过 `vm_xml/` 子包，消除散落的 XML 操作代码。

### Task 4.3 — handler/vm.go 拆分
- 提取 SSE 相关 → `handler/vm_sse.go`
- 提取救援/密码重置 → `handler/vm_rescue.go`

### Task 4.4 — handler/user.go 拆分
- 提取自助 API → `handler/user_self.go`

### Task 4.5 — handler 公共提取
- 跨 handler 请求类型 → `handler/types.go`
- 通用工具函数 → `handler/helpers.go`

**阶段四验证：** `go build ./server/...` 编译通过

---

## 验证方案

每个阶段完成后执行：
1. **编译检查：** `go build ./server/...`
2. **测试运行：** `go test ./server/...`（确保现有测试通过）
3. **功能验证：** 通过 SSH-MCP 连接测试机，确认服务正常启动（等待约 15 秒）
4. **接口验证：** 抽检核心 API（ListVMs, CloneVM, CreateSnapshot）返回正常

---

## 风险与注意事项

1. **循环依赖风险：** clone ↔ template 存在双向调用，需通过接口或将共享逻辑提升到公共层解决
2. **init() 函数：** 某些文件可能含 init() 注册逻辑，移动时需确保包初始化顺序不变
3. **全局变量：** service 包内可能有共享的全局变量（如缓存 map），需统一归属
4. **handler import 变更量大：** 每次拆分子包后 handler 层需同步更新 import，建议用 goimports 自动处理
5. **RPC 策略：** 在拆解 libvirt.go 时，按照已有决策移除 virsh fallback，采用纯 RPC

---

## 预期收益

| 指标 | 改造前 | 改造后 |
|------|--------|--------|
| 单文件最大行数 | 3409 (vpc.go) | < 600 |
| 子包数量 | 0 | 20+ |
| 目录层级 | 1 层平铺 | 3 层（domain → subdomain → file） |
| 跨文件重复逻辑 | IP 解析 4 处、XML 5 处 | 统一入口 |
