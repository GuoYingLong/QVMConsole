# server/service 目录模块化整改计划

## 一、现状诊断

当前 `server/service` 目录包含 **86 个非测试源文件**，其中 8 个文件超过 1200 行：

| 文件 | 行数 | 主要职责 |
|------|------|----------|
| clone.go | 2396 | 虚拟机克隆（单/批量）、重装、Linux/Windows/FnOS 初始化脚本、XML 构造 |
| template.go | 2238 | 模板 CRUD、元数据、磁盘链管理、默认配置、导入导出 |
| libvirt.go | 2080 | VM 类型定义、ListVMs/GetVM、开关机/重启、IP 检测、宿主机统计、XML 解析 |
| snapshot.go | 1930 | 快照 CRUD、内/外部快照、AppArmor、磁盘链管理 |
| network.go | 1774 | 端口转发、静态 IP、VPC 网络、iptables 规则持久化 |
| disk.go | 1598 | 磁盘 CRUD、CD/DVD、IOPS 调优、PCIe 端口管理 |
| vm_dynamic_memory.go | 1244 | 动态内存（balloon/virtio-mem）调度、XML 处理 |
| bandwidth.go | 1213 | 带宽限速（TC/OVS）、用户配额、全局带宽 |

核心问题：
1. **上帝文件** — 单个文件承担过多职责，修改影响面大
2. **职责混杂** — 类型定义、业务逻辑、工具函数共存同一文件
3. **跨域耦合** — 网络/带宽/公网IP 三层代码互相调用底层命令
4. **XML 逻辑散落** — domain XML 生成/解析在 clone.go、vm_dynamic_memory.go、disk.go、libvirt.go 各自实现
5. **初始化脚本硬编码** — clone.go 内嵌大量 virt-customize 命令字符串，难以维护

---

## 二、目标目录结构

```
server/service/
├── vm/                              # ★ 虚拟机核心
│   ├── types.go                     #   VmInfo, VmDetail, VmStats, HostStats, VMListOptions
│   ├── lifecycle.go                 #   StartVM, ShutdownVM, DestroyVM, RebootVM, ResetVM
│   ├── list.go                      #   ListVMs, GetVM, GetVMIPInfo, 缓存相关
│   ├── config.go                    #   EditVMConfig, SetVMBootOrder, SetVMAutostart, SetVMNicModel
│   ├── freeze.go                    #   GetVMFreeze, SetVMFreeze, isVMFreezeEnabled
│   ├── remark.go                    #   GetVMRemark, SetVMRemark
│   ├── owner.go                     #   FindVMOwner, GetUserVMList
│   ├── migration_lock.go            #   EnsureVMNotMigrating, applyVMUnderMigrationStatus
│   ├── runtime.go                   #   VMRuntimeInfo, UpdateVMRuntimeState, 持续运行追踪
│   ├── cache.go                     #   vm_cache.go 现有逻辑
│   ├── credential.go                #   GetVMCredential, SetVMCredential (vm_credential.go)
│   └── config_metadata.go           #   readVMConfigMetadata, writeVMConfigMetadata
│
├── vm_xml/                          # ★ 虚拟机 XML 操作（从各文件提取集中）
│   ├── domain_xml.go                #   ListVMs/GetVM 中的 XML 解析（VmInfo 填充）
│   ├── boot_type.go                 #   ParseVMBootTypeFromDomainXML, ApplyVMBootTypeToDomainXML
│   ├── cpu_xml.go                   #   ParseVCPUCountFromDomainXML, ApplyCPUTopologyMode, ApplyVMCPULimit
│   ├── memory_xml.go                #   ApplyDynamicMemoryConfigToDomainXML, ApplyVirtioMemConfigToDomainXML
│   ├── device_xml.go                #   ParseVMVideoModel, ApplyVMVideoModel, ApplyVMGuestAgent
│   ├── smbios_xml.go                #   ApplySMBIOS1ConfigToDomainXML
│   ├── apic_pae_xml.go             #   ApplyVMAPIC, ApplyVMPAE
│   ├── rtc_xml.go                   #   ApplyRTCConfigToDomainXML
│   ├── vcpu_tag.go                  #   BuildVCPUTag
│   ├── parse_helpers.go             #   parseInfoInt, parseMemStat, parseIfStat, parseBlkStat, parseQemuInfo*
│   └── boot_devices.go              #   parseBootDevices, BootDevice 类型
│
├── network/                         # ★ 网络
│   ├── port_forward/                #   端口转发子模块
│   │   ├── types.go                 #     PortForwardRule, PortForwardAddParams 等
│   │   ├── rules.go                 #     ListPortForwards, AddPortForward, DeletePortForward, UpdatePortForward
│   │   ├── persistence.go           #     SavePortForwardRules, RestorePortForwardRules
│   │   ├── availability.go          #     IsPortAvailable, AutoAllocatePort, getOccupiedPorts
│   │   └── firewall_exemption.go    #     SetPortForwardFirewallExemption, ClearPortForwardFirewallExemption
│   ├── static_ip/                   #   静态 IP 子模块
│   │   ├── ovs.go                   #     OVS 静态 IP 绑定/解绑
│   │   ├── vpc.go                   #     VPC 静态 IP 绑定/解绑
│   │   └── auto.go                  #     findFreeIP, EnsureStaticIP, BindStaticIP
│   ├── vpc/                         #   VPC 网络（从 bandwidth.go / network.go 提取）
│   │   └── switch.go                #     getVPCSwitchForVM, ApplyVPCSwitchRuntime
│   └── diagnostics/                 #   网络诊断
│       └── diagnostics.go           #     network_diagnostics.go 现有逻辑
│
├── bandwidth/                       # ★ 带宽管理
│   ├── types.go                     #   BandwidthInfo, BandwidthDetail
│   ├── tc.go                        #   TC 下行/上行（applyTCDownloadLimit, applyTCUploadLimit, IFB）
│   ├── ovs.go                       #   OVS 流表/队列/meter（applyOVSBandwidthLimit, clearOVSBandwidthLimit）
│   ├── vm.go                        #   ApplyVMBandwidth, ClearVMBandwidth, GetVMBandwidth
│   ├── quota.go                     #   RebalanceUserBandwidth, SetVMCustomAverage（用户配额）
│   └── global.go                    #   ApplyGlobalBandwidthLimit, ClearGlobalBandwidthLimit
│
├── clone/                           # ★ 克隆/创建
│   ├── types.go                     #   CloneParams, BatchCloneParams, ReinstallParams, CloneResult
│   ├── clone.go                     #   CloneVM 主逻辑（精简后）
│   ├── batch_clone.go               #   BatchCloneVM（从 clone.go 后半部分提取）
│   ├── reinstall.go                 #   ReinstallVM（从 clone.go 提取）
│   ├── init_linux.go                #   initLinuxClone, prepareLinuxCloneFirstBootIdentity
│   ├── init_windows.go              #   cloneWindows, buildWindowsUnattendXML, injectWindowsUnattendFile
│   ├── init_fnos.go                 #   cloneFnOS, FnOS 初始化命令
│   └── xml_pipeline.go              #   CloneVM 内 XML 注入管线（ApplyRTC, ApplySMBIOS1, ApplyVMGuestAgent 等串联）
│
├── template/                        # ★ 模板
│   ├── types.go                     #   模板类型定义（复用 template.go 中现有类型）
│   ├── crud.go                      #   模板 CRUD
│   ├── meta.go                      #   GetTemplateMeta, WriteTemplateMeta（从 template.go 提取）
│   ├── chain.go                     #   磁盘链管理/验证
│   ├── config.go                    #   默认配置（template_default_config_test 对应功能）
│   ├── detection.go                 #   DetectTemplateBootType, detectTemplateDisk
│   └── transfer/                    #   模板传输
│       └── transfer.go              #     template_transfer.go 现有逻辑
│
├── snapshot/                        # ★ 快照
│   ├── types.go                     #   SnapshotInfo, SnapshotQuotaInfo, 相关结构体
│   ├── core.go                      #   CreateSnapshot, DeleteSnapshot, RevertSnapshot, ListSnapshots
│   ├── external.go                  #   外部快照操作（revertExternalSnapshot, deleteExternalSnapshot）
│   ├── chain.go                     #   磁盘链合并（commitActiveExternalOverlay, commitInactiveExternalOverlay）
│   ├── nvram.go                     #   UEFI NVRAM 处理
│   ├── apparmor.go                  #   AppArmor 规则管理
│   ├── quota.go                     #   快照配额检查
│   └── cleanup.go                   #   残留清理
│
├── public_ip/                       # ★ 公网 IP
│   ├── types.go                     #   PublicIPRequest, PublicIPBindRequest, PublicIPInfo
│   ├── crud.go                      #   ListPublicIPs, CreatePublicIP, UpdatePublicIP, DeletePublicIP
│   ├── binding.go                   #   bindPublicIP, unbindPublicIP, migratePublicIP
│   ├── nat.go                       #   1:1 NAT 规则
│   ├── classic.go                   #   经典网络（路由/桥接）
│   └── rules.go                     #   ApplyPublicIPRules, BuildPublicIPRulesScript
│
├── ip_resolver/                     # ★ IP 解析（从 libvirt.go 1700+ 行中提取，网络/带宽/公网IP共用）
│   ├── resolver.go                  #   getVMIP（主动/被动探测统一入口）
│   ├── arp.go                       #   ARP 表扫描、主动 arp-scan/nmap
│   ├── vpc.go                       #   VPC 相关的 IP 查找（GetVPCLeaseIPForVM 等）
│   └── neighbor.go                  #   ip neigh 邻居表查询
│
├── storage/                         # ★ 存储
│   ├── disk/
│   │   ├── types.go                 #     DiskInfo, DiskSimpleInfo, DiskIOPSTune
│   │   ├── crud.go                  #     ListDisks, AddDisk, RemoveDisk, ResizeDisk
│   │   ├── cdrom.go                 #     ChangeCDROM, EjectCDROM, RemoveCDROM
│   │   ├── iops.go                  #     SetDiskIOPSTune, GetDiskIOPSTune, ParseAllDiskIOPSTune
│   │   └── pcie.go                  #     SetVMPCIERootPorts, GetVMPCIERootPorts, 热插拔 PCIe
│   └── pool/
│       └── pool.go                  #     storage_pool.go 现有逻辑
│
├── firewall/                        # ★ 防火墙
│   ├── policy.go                    #   防火墙策略（firewall.go）
│   ├── host.go                      #   宿主机防火墙（host_firewall.go）
│   └── ufw.go                       #   UFW 操作
│
├── security/                        # ★ 安全（account/2FA/challenge 等，已基本独立）
│   ├── account.go                   #   security_account.go
│   ├── challenge.go                 #   security_challenge.go
│   ├── crypto.go                    #   security_crypto.go
│   ├── smtp.go                      #   security_smtp.go
│   ├── token.go                     #   security_token.go
│   ├── totp.go                      #   security_totp.go
│   ├── user.go                      #   security_user.go
│   └── constants.go                 #   security_constants.go
│
├── libvirt_rpc/                     # ★ Libvirt RPC 封装（已提取）
│   ├── libvirt.go                   #   GetLibvirt, IsLibvirtRPCAvailable
│   ├── libvirt_rpc.go               #   连接管理
│   └── libvirt_rpc_domain.go        #   领域操作（Domain RPC 方法）
│
├── lightwei...（内容过长截断，见完整文件）ght/                        # ★ 轻量云
│   ├── cloud.go                     #   lightweight_cloud.go
│   ├── registration.go              #   lightweight_vm_registration.go
│   └── quota.go                     #   lightweight_runtime_quota.go, user_runtime_quota.go
│
├── user/                            # ★ 用户
│   ├── core.go                      #   user.go
│   ├── quota.go                     #   user_quota.go
│   ├── storage.go                   #   user_storage.go
│   └── ssh.go                       #   user_ssh.go
│
├── host/                            # ★ 宿主机管理
│   ├── stats.go                     #   GetHostStats, stats_collector, host_stats_disk
│   ├── node.go                      #   host_node.go
│   ├── ksm.go                       #   host_ksm.go
│   ├── zram.go                      #   host_zram.go
│   └── maintenance.go               #   maintenance.go
│
├── scheduler/                       # ★ 调度器
│   └── center.go                    #   scheduler_center.go
│
├── traffic/                         # ★ 流量配额
│   └── quota.go                     #   traffic_quota.go
│
├── ovs/                             # ★ OVS 相关
│   ├── network.go                   #   ovs_network.go
│   └── diagnostics.go              #   ovs_diagnostics.go
│
├── rescue/                          # ★ 救援模式
│   └── rescue.go                    #   rescue.go
│
├── resource_check/                  # ★ 资源检查
│   └── check.go                     #   resource_check.go
│
├── remote_exec/                     # ★ 远程执行
│   └── exec.go                      #   remote_exec.go
│
├── share/                           # ★ 共享目录
│   └── share.go                     #   share.go
│
├── vnc/                             # ★ VNC
│   └── vnc.go                       #   vnc.go
│
├── kvm_module/                      # ★ KVM 模块
│   └── module.go                    #   kvm_module.go
│
├── api_key/                         # ★ API Key
│   └── api_key.go                   #   api_key.go
│
├── quota_fs/                        # ★ 配额文件系统
│   └── quota_fs.go                  #   quota_fs.go
│
├── jwt_secret/                      # ★ JWT
│   └── secret.go                    #   jwt_secret.go
│
└── stats_collector/                 # ★ 统计采集
    └── collector.go                 #   stats_collector.go
```

---

## 三、分阶段执行计划

### 阶段一：提取共用基础模块（优先级最高，无风险）

| 序号 | 任务 | 源文件 | 目标 |
|------|------|--------|------|
| 1.1 | 提取 VM 类型定义 | libvirt.go | `vm/types.go` |
| 1.2 | 提取 IP 解析逻辑 | libvirt.go | `ip_resolver/` |
| 1.3 | 提取 XML 解析/生成 | libvirt.go, clone.go, disk.go, vm_dynamic_memory.go | `vm_xml/` |
| 1.4 | 提取带宽类型定义 | bandwidth.go | `bandwidth/types.go` |

**理由**：类型和工具代码不涉及业务逻辑变更，提取后所有文件 `import` 新包即可，无副作用。`ip_resolver` 会被 network/bandwidth/public_ip 三处引用，集中后消除重复。

---

### 阶段二：拆解上帝文件（中风险，需回归测试）

| 序号 | 任务 | 源文件 | 目标 |
|------|------|--------|------|
| 2.1 | 拆解 libvirt.go（2080行 → 5个文件） | libvirt.go | `vm/lifecycle.go`, `vm/list.go`, `vm/config.go`, `ip_resolver/`, `host/stats.go` |
| 2.2 | 拆解 clone.go（2396行 → 8个文件） | clone.go | `clone/clone.go`, `clone/batch_clone.go`, `clone/reinstall.go`, `clone/init_linux.go`, `clone/init_windows.go`, `clone/init_fnos.go`, `clone/xml_pipeline.go` |
| 2.3 | 拆解 snapshot.go（1930行 → 8个文件） | snapshot.go | `snapshot/core.go`, `snapshot/external.go`, `snapshot/chain.go`, `snapshot/nvram.go`, `snapshot/apparmor.go`, `snapshot/quota.go`, `snapshot/cleanup.go` |
| 2.4 | 拆解 network.go（1774行 → 子模块） | network.go | `network/port_forward/`, `network/static_ip/` |
| 2.5 | 拆解 disk.go（1598行 → 子模块） | disk.go | `storage/disk/crud.go`, `storage/disk/cdrom.go`, `storage/disk/iops.go`, `storage/disk/pcie.go` |
| 2.6 | 拆解 vm_dynamic_memory.go（1244行） | vm_dynamic_memory.go | `vm_xml/memory_xml.go` + 保留调度器在 `vm/` 下 |

---

### 阶段三：提取功能子包（低风险，增强内聚）

| 序号 | 任务 | 说明 |
|------|------|------|
| 3.1 | 带宽子包 | `bandwidth/` 已有充分边界，提取 TC/OVS 实现细节 |
| 3.2 | 公网 IP 子包 | `public_ip/` 独立性强，几乎无外部依赖 |
| 3.3 | 防火墙子包 | `firewall/` 合并 policy + host + ufw |
| 3.4 | 安全子包整理 | `security/` 已基本独立，仅做目录归并 |
| 3.5 | 轻量云子包整理 | `lightweight/` 合并 cloud + registration + quota |
| 3.6 | 用户子包整理 | `user/` 合并 core + quota + storage + ssh |

---

### 阶段四：消除重复逻辑

| 序号 | 任务 | 说明 |
|------|------|------|
| 4.1 | 统一 IP 获取 | 确保 network/bandwidth/public_ip/clone 均通过 `ip_resolver` 获取 IP |
| 4.2 | 统一 XML 操作 | 确保 clone/vm_xml/vm_dynamic_memory 中 XML 修改均通过 `vm_xml` 子包 |
| 4.3 | 统一磁盘操作 | disk 子包收口所有磁盘路径获取/修改逻辑 |

---

## 四、高频调用函数提取清单

以下函数在多个模块间被高频交叉调用，建议优先提取到独立子包：

| 函数 | 当前所在文件 | 调用方 |
|------|-------------|--------|
| `getFirstVMMAC` / `getVMMAC` | libvirt.go, bandwidth.go | bandwidth, network, public_ip, clone |
| `getVMIP` | libvirt.go | libvirt, clone, network |
| `getDomainStateRPC` | libvirt_rpc_domain.go | libvirt, clone, disk, snapshot, bandwidth |
| `getDomainXMLRPC` | libvirt_rpc_domain.go | libvirt, clone, disk, snapshot, vm_dynamic_memory |
| `parseDisksFromDomainXML` | vm_xml 相关 | libvirt, disk, clone |
| `GetOVSStaticHostByVMName` | ovs_network.go | network, bandwidth, public_ip |
| `getVPCSwitchForVM` | bandwidth.go | bandwidth, network, public_ip |
| `IsLightweightCloudVM` | lightweight_cloud.go | libvirt, bandwidth, clone, snapshot |
| `MbpsToKBps` / `KBpsToMbps` | bandwidth.go | bandwidth, clone |
| `NormalizeVMDiskBus` | disk.go | disk, clone |

这些函数建议统一归属到：
- `getFirstVMMAC` / `getVMIP` → `ip_resolver/`
- XML 解析函数 → `vm_xml/parse_helpers.go`
- `getVPCSwitchForVM` → `network/vpc/`
- `MbpsToKBps` → `bandwidth/types.go`

---

## 四-B、handler 目录分析

`server/handler` 共 34 个文件，整体结构比 service 层好，但仍有改进空间：

| 文件 | 行数 | 主要职责 | 问题 |
|------|------|----------|------|
| vm.go | 1388 | VM 列表/详情/操作/编辑/SSE/救援/密码重置 | **偏大**，可拆分 |
| user.go | 1164 | 用户 CRUD、配额、VM分配、自助 API | **偏大**，自助部分可独立 |
| auth.go | 997 | 登录/2FA/邮箱/找回密码/邀请注册 | 目前结构合理 |
| settings.go | 705 | 系统设置读写 | 结构合理 |
| clone.go | 640 | 克隆/批量克隆/重装/删除 | 结构合理 |
| network.go | 1035 | 端口转发/静态IP/VPC路由 | **偏大**，可拆分 |
| user_storage.go | 671 | 用户存储管理 | 结构合理 |
| disk.go | 627 | 磁盘操作 | 结构合理 |

**handler 层的改进建议**（优先级低，可在 service 改造后跟进）：

| 序号 | 任务 | 说明 |
|------|------|------|
| H1 | vm.go 拆分 | 将 SSE 相关（GetVmListSSE, GetVmDetailSSE）提取到 `handler/vm_sse.go`；救援/密码重置保留在 vm.go 或拆到 `handler/vm_rescue.go` |
| H2 | user.go 拆分 | 将自助 API（GetSelfVMs, SelfCloneVm, SelfDeleteVm 等）提取到 `handler/user_self.go` |
| H3 | handler 类型集中 | 将 `VmEditRequest`, `CloneVmRequest` 等跨 handler 使用的请求类型放到 `handler/types.go`，消除 handler 间重复定义（如 `SelfCloneVmRequest` 与 `CloneVmRequest` 几乎相同） |
| H4 | 通用工具函数 | 将 `parseBoolQuery`, `buildBaseURL`, `getCurrentUser`, `requireHighRiskVerification` 等跨 handler 复用的函数提取到 `handler/helpers.go` |

---

## 四-C、middleware 目录分析

`server/middleware` 共 4 个文件，规模小且职责清晰：

| 文件 | 行数 | 职责 |
|------|------|------|
| auth.go | 323 | JWT 生成/解析、多种 Token 类型中间件、API Key 认证、权限/VM 访问控制 |
| ratelimit.go | 210 | API 限频（令牌桶模型） |
| cors.go | 23 | CORS 跨域处理 |
| request_logger.go | 69 | 请求日志 |

**middleware 评估**：当前结构良好，无需大规模整改。唯一建议是 `auth.go` 可在未来超过 500 行后拆为 `auth_jwt.go`（JWT 操作）和 `auth_middleware.go`（中间件链）。

---

## 四-D、router 目录分析

`server/router/router.go` 单文件 528 行，路由定义清晰：

- 按功能分组（vm / template / network / vpc / firewall / ovs / storage-pool / host / task / scheduler / user / self）
- 中间件挂载明确（AdminMiddleware, ElasticCloudOnlyMiddleware, VMAccessMiddleware 等）
- 静态文件服务逻辑（SPA fallback）独立于路由定义

**router 评估**：当前结构良好。如果未来路由超过 800 行，可考虑按功能拆为 `router/vm.go`, `router/network.go` 等子文件，但非必须。

---

## 四-E、config 目录分析

`server/config/config.go` 单文件 804 行，包含：
- 全局配置结构体 `Config`（约 80 个字段）
- 环境变量加载（`LoadConfig`）
- 默认值定义
- `.env` 文件同步（`SyncEnvFile`, `writeEnvFile`）
- 配置项转 Map（`ToSettingsMap`）
- 多个 getter 函数（如 `DefaultMaintenanceServiceUnits`）

**config 评估**：单文件偏大但可接受，因为配置项是平面属性。如需拆分建议：
- `config/types.go` — Config 结构体 + 默认值
- `config/loader.go` — LoadConfig / 环境变量解析
- `config/env.go` — .env 文件读写

---

## 四-F、model 目录分析

`server/model` 共 27 个文件，大部分是 GORM 模型定义，规模小。其中：

| 文件 | 行数 | 职责 |
|------|------|------|
| db.go | 279 | 数据库初始化、迁移 |
| scheduler_event.go | 160 | 调度事件模型 |
| vpc.go | 113 | VPC 模型 |
| vm_schedule.go | 109 | VM 调度模型 |

**model 评估**：结构良好，每个模型独立文件。`db.go` 包含数据库初始化和全局 DB 变量，与模型定义耦合，可考虑将 DB 连接管理独立到 `model/db.go` 保留，但模型定义可按业务域分文件夹（如 `model/vm/`, `model/network/`），非优先事项。

---

## 四-G、taskqueue 目录分析

`server/taskqueue` 仅 2 个文件：

| 文件 | 行数 | 职责 |
|------|------|------|
| queue.go | 561 | 任务队列核心：提交/执行/取消/SSE推送/重试 |
| queue_test.go | 117 | 测试 |

**taskqueue 评估**：当前单文件 561 行可接受。如果未来扩展，可考虑拆为 `queue_core.go`（提交/执行）、`queue_sse.go`（SSE 推送）、`queue_retry.go`（重试逻辑）。

---

## 四-H、utils 目录分析

`server/utils` 仅 6 个小文件：

| 文件 | 行数 | 职责 |
|------|------|------|
| cmd.go | 154 | 命令行执行（ExecCommand, ExecShell, 超时控制） |
| cmd_process_unix.go | 24 | Unix 进程组管理 |
| cmd_process_windows.go | 15 | Windows 进程管理 |
| fs.go | 81 | 文件系统操作（ReadMemInfo, GetDiskSpace, ChownLibvirtQEMU） |
| fs_linux.go | 68 | Linux 文件系统扩展 |

**utils 评估**：结构精简，职责明确（命令执行 + 文件系统），无需整改。

---

## 四-I、其他顶层文件

| 文件 | 行数 | 职责 |
|------|------|------|
| main.go | - | 服务入口、初始化顺序（config → model → scheduler → taskqueue → router） |
| go.mod | - | Go 模块定义 |
| logger/logger.go | - | 日志（libvirt/app 双通道） |
| logger/rotation.go | - | 日志轮转 |

**评估**：结构合理，无需整改。

---

## 四-J、目录整改优先级总览

| 优先级 | 目录 | 问题程度 | 建议动作 |
|--------|------|----------|----------|
| ★★★ 最高 | service/ | 严重：86 个平铺文件，8 个 1200+ 行 | 按本文第二阶段方案全面重构 |
| ★★☆ 中 | handler/ | 一般：vm.go(1388行)/user.go(1164行) 偏大 | service 改造后跟进拆分 |
| ★☆☆ 低 | config/ | 轻微：804 行单文件 | 可拆分为 types/env/loader |
| ☆☆☆ 无 | middleware/ | 良好 | 暂无需整改 |
| ☆☆☆ 无 | router/ | 良好 | 暂无需整改 |
| ☆☆☆ 无 | model/ | 良好 | 暂无需整改 |
| ☆☆☆ 无 | taskqueue/ | 良好 | 暂无需整改 |
| ☆☆☆ 无 | utils/ | 良好 | 暂无需整改 |
| ☆☆☆ 无 | logger/ | 良好 | 暂无需整改 |

---

## 五、注意事项

1. **所有内部函数保持 package 内可见（小写）**，仅对外 API 使用大写导出
2. **每个子包包含自己的 `_test.go`**，移动时同步迁移测试文件
3. **handler 层无需变更**，仅修改 service 层 import 路径
4. **不改变任何函数签名和外部行为**，确保前端/API Key 调用不受影响
5. **libvirt_rpc/ 子包已初步提取**，阶段一即可完成剩余 RPC 方法迁移
6. **vm_xml/ 子包最为关键**，集中后 clone.go 可减少约 400 行 XML 操作代码
7. **分批提交**，每完成一个阶段提交一次，便于 code review 和回滚

---

## 六、预期收益

| 指标 | 改造前 | 改造后 |
|------|--------|--------|
| 单文件最大行数 | 2396 (clone.go) | < 800 |
| 平均文件行数 | ~450 | ~200 |
| 目录层级 | 1 层（全部平铺） | 3 层（domain → subdomain → file） |
| 跨文件重复逻辑 | IP 解析 4 处、XML 解析 5 处 | 统一入口 |
| 新人理解成本 | 需阅读 2000+ 行才能修改 | 按目录名即可定位 |
