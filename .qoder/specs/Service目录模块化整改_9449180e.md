# Service 目录模块化整改执行计划

## Context

`server/service` 根目录当前仍有约 100 个 `.go` 文件未归入子目录。已有的模块化框架（delegate/register/compat 三文件 + Hook 注入）运作良好，但 VM 相关 35 个文件、template.go（2248行）等"上帝文件"严重影响代码可维护性。本计划基于已有骨架"补齐"，而非推翻重来。

---

## P0：VM 35 个文件整合入 `vm/` 子包（最高优先级）

**目标**：将根目录 35 个 `vm_*.go` 文件迁入已有的 `service/vm/` 子目录，使 VM 相关代码完全内聚。

### 执行策略

采用与 `clone/`、`snapshot/` 相同的三文件胶水模式：
- 新建 `vm/deps.go` — Deps 结构体，承载所有外部依赖（snapshot/clone/bandwidth/network 等）
- 根目录保留 `vm_delegate.go` + `vm_register.go` + `vm_compat.go`

### 文件迁移映射

| 源文件（根目录） | 目标位置（vm/） | 说明 |
|---|---|---|
| vm_types.go | vm/types.go | 核心类型 VmInfo/VmDetail/VmStats 等 |
| vm_lifecycle.go | vm/lifecycle.go | StartVM/ShutdownVM/DestroyVM/RebootVM |
| vm_list.go | vm/list.go | ListVMs/GetVM |
| vm_detail.go | vm/detail.go | GetVMDetail |
| vm_create.go | vm/create.go | CreateVM |
| vm_config.go | vm/config.go | EditVMConfig |
| vm_config_metadata.go | vm/config_metadata.go | readVMConfigMetadata |
| vm_cache.go | vm/cache.go | VM缓存 |
| vm_credential.go | vm/credential.go | 凭据管理 |
| vm_runtime.go | vm/runtime.go | 运行时状态 |
| vm_stats.go | vm/stats.go | 统计信息 |
| vm_freeze_remark.go | vm/freeze_remark.go | 冻结/备注 |
| vm_shared_helpers.go | vm/helpers.go | DomainExists/GetDomainState |
| vm_interface.go | vm/interface.go | 网卡操作 |
| vm_xml.go | vm/xml.go | XML 操作入口 |
| vm_display.go | vm/display.go | 显示配置 |
| vm_boot_type.go | vm/boot_type.go | 启动类型 |
| vm_clock.go | vm/clock.go | RTC 时钟 |
| vm_apic.go | vm/apic.go | APIC 配置 |
| vm_pae.go | vm/pae.go | PAE 配置 |
| vm_smbios.go | vm/smbios.go | SMBIOS 配置 |
| vm_cpu_affinity.go | vm/cpu_affinity.go | CPU 亲和性 |
| vm_cpu_limit.go | vm/cpu_limit.go | CPU 限制 |
| vm_cpu_topology.go | vm/cpu_topology.go | CPU 拓扑 |
| vm_passthrough.go | vm/passthrough.go | 设备直通 |
| vm_guest_agent.go | vm/guest_agent.go | Guest Agent |
| vm_install_media.go | vm/install_media.go | 安装介质 |
| vm_disk_migration.go | vm/disk_migration.go | 磁盘迁移 |
| vm_first_boot.go | vm/first_boot.go | 首次启动 |
| vm_password_reset.go | vm/password_reset.go | 密码重置 |
| vm_name.go | vm/name.go | 名称生成 |
| vm_lock.go | vm/lock.go | 锁定 |
| vm_monitor.go | vm/monitor.go | QEMU Monitor |
| vm_schedule.go | vm/schedule.go | 定时任务 |
| vm_export.go | vm/export.go | 导出 |

### 新建文件

```
vm/deps.go — Deps 结构体（参照 clone/deps.go 模式）
```

### 根目录保留文件（胶水层）

```
vm_delegate.go   — 转发 handler 调用 → vm 子包
vm_register.go   — init() 注入 Deps（snapshot/clone/bandwidth 等回调）
vm_compat.go     — type VmInfo = vm.VmInfo 等别名
```

### 关键约束

1. **vm_shared_helpers.go 中的 `DomainExists`、`GetDomainState`、`QemuInfoChain`** 被 clone/snapshot/bandwidth 等多个子包调用 → 必须通过 Hook 注入或保持在 vm/ 中导出
2. **vm_types.go** 中的类型被 handler 层大量引用 → 通过 compat.go 类型别名保持 `service.VmInfo` 可用
3. 已有的 `vm/memory/`、`vm/migration/`、`vm/vmimport/` 保持不动

---

## P1：Template + Network Bridge + Storage Pool（次高优先级）

### Task 1.1：template.go → `template/` 子包

**规模**：2248 行 → 拆为 6-7 个文件

| 目标文件 | 内容 |
|---|---|
| template/types.go | 类型定义 |
| template/crud.go | CRUD 操作 |
| template/meta.go | 元数据读写 |
| template/chain.go | 磁盘链管理 |
| template/config.go | 默认配置 |
| template/detection.go | 启动类型检测 |
| template/deps.go | Hook 依赖 |

根目录保留：`template_delegate.go`（已有 `template.go` 需重命名）+ `template_compat.go`

### Task 1.2：network_bridge.go → `network/bridge.go`

- 743 行迁入 `network/` 子目录
- 在 `network/deps.go` 中补充必要 Hook
- 根目录保留对应 delegate 函数（合并到现有 `network_register.go`）

### Task 1.3：storage_pool.go → `storage/pool/`

- 已有 `storage/disk/`，补建 `storage/pool/pool.go`
- 根目录保留 delegate

### Task 1.4：network_diagnostics.go + port_forward_probe.go → `network/`

- 归入已有的 `network/` 子目录

---

## P2：胶水文件精简（中优先级）

**原则**：不移入子目录，但合并减少文件数。

### 合并规则

每个模块的 `_delegate.go` + `_register.go` + `_compat.go` 三文件合并为一个 `_wire.go`：

| 当前 | 合并后 |
|---|---|
| ovs_delegate.go + ovs_register.go + ovs_compat.go | ovs_wire.go |
| bandwidth_delegate.go + bandwidth_register.go | bandwidth_wire.go |
| vnc_delegate.go + vnc_register.go + vnc_compat.go | vnc_wire.go |
| ... | ... |

**预期效果**：43 个胶水文件 → ~16 个 wire 文件

### 不合并的例外

- `clone_delegate.go`（140行）+ `clone_export.go`：clone 子包依赖复杂，保持分离
- `snapshot_register.go`：依赖注入逻辑复杂

---

## P3：其他独立文件归位（低优先级）

| 文件 | 目标 | 理由 |
|---|---|---|
| stats_collector.go (357行) | host/ | 与宿主机统计内聚 |
| resource_check.go (152行) | host/ | 资源检查属宿主机职责 |
| linked_clone.go (432行) | clone/ | 克隆变体 |
| jwt_secret.go (131行) | security/ | 安全相关 |
| maintenance_helper.go | host/ | 维护模式 |
| quota_fs.go (438行) | 保持根目录或 storage/ | 跨模块使用，按需决定 |
| remote_exec.go (219行) | 保持根目录 | 通用工具 |
| kvm_module.go (240行) | 保持根目录 | 全局初始化 |
| api_key.go (178行) | 保持根目录 | 边界清晰但体量小 |

---

## 执行顺序与依赖

```
P0 (VM) ──→ P1.1 (Template，依赖 VM 的 Hook 稳定)
         ├─→ P1.2 (Network Bridge，独立)
         ├─→ P1.3 (Storage Pool，独立)
         └─→ P1.4 (Network散落文件，独立)

P1 完成后 ──→ P2 (胶水精简，需所有子包稳定)
P2 完成后 ──→ P3 (低优先级归位)
```

---

## 关键风险与缓解

| 风险 | 缓解措施 |
|---|---|
| 循环 import | 严格遵循单向依赖：root → 子包，子包间通过 Hook 通信 |
| handler 层大量 import service.XxxType | 使用 compat.go 类型别名，handler 零改动 |
| vm_shared_helpers 被多包调用 | 在 vm/ 中导出，其他包通过 Hook 或直接 import vm 包 |
| template 与 clone/snapshot 循环依赖 | template 子包不 import clone/snapshot，反向通过 Hook |
| 合并胶水文件时遗漏 init() 注册 | 逐文件对比，确保所有 Hook 赋值保留 |

---

## 验证方式

1. **编译验证**：每个 Task 完成后 `go build ./...` 确保无编译错误
2. **功能验证**：测试环境自动热重载（文件同步至 /opt/project/QVMConsole），验证：
   - VM 列表/详情/生命周期操作正常
   - 克隆/快照功能不受影响
   - 网络端口转发/静态IP正常
3. **回归验证**：通过 SSH MCP 在测试机上执行核心操作路径

---

## 关键文件路径

- 已有模块化计划：`/server-service-modularization-plan.md`
- VM 子目录：`/server/service/vm/`（memory/, migration/, vmimport/）
- 典型胶水模式参考：`/server/service/ovs_delegate.go` + `ovs_register.go` + `ovs_compat.go`
- Clone 依赖注入参考：`/server/service/clone/deps.go`
- Network 依赖注入参考：`/server/service/network/deps.go`
