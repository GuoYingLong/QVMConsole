# 虚拟机创建增强 & 基础设施管理

## 功能概述

本次更新新增和增强了以下功能：

### 1. 创建虚拟机增强

#### 三种创建模式
- **模板克隆**：通过已有模板克隆创建（原有功能）
- **原生链式克隆**：仅管理员可见，直接基于模板创建 backing qcow2 后启动虚拟机，不执行任何来宾初始化
- **普通创建**：不通过模板，从头创建虚拟机

#### 普通创建支持的配置项

| 配置项 | 说明 | 选项 |
|--------|------|------|
| 虚拟化方案 | 虚拟化技术选择 | 硬件虚拟化 KVM（推荐） / 软件虚拟化 QEMU |
| 平台架构 | 目标 CPU 架构（软件虚拟化时可选） | x86_64（默认） / aarch64 (ARM) / riscv64 (RISC-V) |
| 系统类型 | 选择操作系统家族 | Linux / Windows |
| 系统版本 | OS 变体（可搜索） | 从 `virt-install --osinfo list` 获取 |
| 虚拟机类型 | 芯片组类型 | Q35（推荐） / i440FX / virt（ARM/RISC-V） |
| 引导类型 | 固件引导方式 | BIOS / UEFI / UEFI + 安全引导 |
| 磁盘大小 | 虚拟磁盘容量 | 最小 10GB (Linux) / 20GB (Windows) |
| 磁盘格式 | 磁盘镜像格式 | QCOW2（推荐） / RAW |
| 磁盘驱动类型 | 磁盘总线控制器 | VirtIO（推荐/高性能） / SCSI / SATA（Windows 兼容） / IDE（传统） |
| 额外磁盘 | 可添加多块数据盘并选择落盘存储位置 | 大小 / 格式 / 总线 / 存储位置 |
| ISO 镜像 | 安装介质 | 从存储池自动扫描 |
| 网络接入 | 网络连接 | 创建、克隆、导入时统一接入当前默认网络，并可额外选择 VPC 交换机/安全组 |
| 网卡类型 | 网络接口卡型号 | VirtIO（推荐/高性能） / e1000e (Intel) / rtl8139（传统兼容） |
| 启动顺序 | 引导设备优先级 | 硬盘 / 光驱 / 网络(PXE)，可排序 |
| Watchdog | 监督者设备 | 不启用 / i6300esb / iTCO |
| 开机自启 | 宿主机启动后自动启动 | 开关 |
| 启动时冻结 CPU | 启动后保持暂停态，适合调试 QEMU/KVM | 开关（freeze） |
| PAE | 控制是否暴露 PAE（物理地址扩展） | 开关（`pae`） |
| RTC 配置 | 控制 RTC 时间基准与起始时间 | 弹窗配置（`rtc_offset` / `rtc_startdate`） |
| SMBIOS 类型 1 | 设置厂商、产品、序列号、UUID 等身份字段 | 弹窗配置（`smbios1`） |

#### ISO 智能识别
选择 ISO 镜像后自动：
- 补全系统类型（Linux/Windows）
- 补全系统版本（如 ubuntu24.04、win11）
- 设置最小磁盘大小（Windows ≥ 20G，Linux ≥ 10G）
- Windows 默认推荐 UEFI + Q35，磁盘驱动默认推荐为 SATA，网卡默认推荐为 e1000e；若用户已经手动改过引导类型，将保留用户选择
- 启动顺序自动调整为光驱优先

#### 编辑模式增强
- 已有磁盘可直接修改驱动类型（VirtIO / SCSI / SATA / IDE），需关机
- 新增磁盘可选择落盘存储位置；普通用户新增容量会计入硬盘配额
- 支持挂载已有磁盘文件（指定路径和总线类型）
- 网卡类型可修改（VirtIO / e1000e / rtl8139），需关机

### 2. 网络入口调整

左侧「虚拟网络」菜单与对应接口已移除。当前网络能力统一收敛到：

- 管理员左侧「网络」
- 普通用户左侧「VPC 网络」
- 虚拟机详情页「网络管理」

### 3. 存储池管理

侧边栏「存储池」菜单用于管理宿主机物理硬盘和分区，不再创建或删除 libvirt storage pool。

#### 功能列表
- 查看宿主机所有硬盘、分区、容量、文件系统、挂载点和使用率
- 设置用户侧显示名称
- 启用或禁用普通用户可选存储位置
- 设置默认虚拟机硬盘位置
- 格式化并挂载硬盘，自动写入 `/etc/fstab`
- 创建虚拟机和模板克隆时选择落盘存储池

#### API 端点

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/storage-pool/list` | 获取宿主机硬盘列表 |
| GET | `/api/storage-pool/vm-targets` | 获取创建虚拟机可选存储位置 |
| GET | `/api/storage-pool/all-isos` | 获取全局 ISO（聚合） |
| GET | `/api/storage-pool/:id` | 获取硬盘详情 |
| PUT | `/api/storage-pool/:id/config` | 更新显示名和启用状态 |
| POST | `/api/storage-pool/:id/default` | 设置默认存储位置 |
| POST | `/api/storage-pool/:id/format-mount` | 格式化并挂载硬盘 |

### 4. 模板管理

侧边栏新增「模板管理」菜单。
- 按模板树展示模板链路，支持根节点和分支节点
- 首个模板生成 `template_uid`，每个节点生成 `node_id`
- 从模板创建 VM 后再次制作模板，会自动挂载到来源节点下
- 管理员只可编辑启用状态、管理员名称和用户侧显示文本
- 普通用户只能从已启用节点克隆，管理员可从任意节点克隆
- 节点展示直接创建 VM 数量和子树 VM 总数
- 导出统一生成 `.tar.gz` 模板包，根节点可导出整棵树，任意节点可导出子树
- 导入先展示完整链路和哈希信息，确认后再导入
- 删除模板前会展示将删除的模板子树和关联虚拟机

#### API 端点

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/template/list` | 获取模板列表（含 `has_meta` 字段） |
| POST | `/api/template/import/preview` | 预览模板包导入链路 |
| POST | `/api/template/import/confirm` | 确认导入模板包（异步任务） |
| GET | `/api/template/:name/vms` | 获取模板关联虚拟机列表 |
| POST | `/api/template/prepare` | 制作模板（异步任务） |
| POST | `/api/template/:name/export?scope=node` | 导出当前节点子树（异步任务） |
| POST | `/api/template/:name/export?scope=root` | 导出整棵模板树（异步任务） |
| DELETE | `/api/template/:name/export` | 删除模板导出文件 |
| GET | `/api/template/download/:filename` | 下载模板导出文件 |
| PUT | `/api/template/:name/publish` | 更新启用状态、显示文本和默认创建配置 |
| GET | `/api/template/:name/delete-preview` | 获取模板链路删除预览 |
| DELETE | `/api/template/:name` | 删除模板（可联动删除关联虚拟机） |

### 5. 系统设置

侧边栏新增「系统设置」菜单。

#### 可配置项
- **端口自动分配范围**：设置端口转发自动分配的起止范围（默认 10000-20000）
- **模板目录**：模板存放位置（默认 `/var/lib/libvirt/images/templates`）
- **模板导入临时目录**：模板上传与解压的临时目录（默认 `<模板目录>/_imports`）
- **模板导出目录**：模板导出文件暂存目录（默认 `<模板目录>/_exports`）
- **克隆磁盘目录**：克隆磁盘存放位置（默认 `/var/lib/libvirt/images`）
- **ISO 存放位置**：创建虚拟机和救援系统选择器读取的全局 ISO 目录（默认 `/var/lib/libvirt/images/ISO`）
- **默认网络**：创建、克隆、导入虚拟机时使用的默认网络名称
- **网段前缀**：内网 IP 段前缀
- **KVM Unrestricted Guest**：Intel KVM 宿主机级兼容性开关，用于 VMware 嵌套虚拟化下的 `hardware error 0x7` 排障

#### API 端点

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/settings` | 获取设置 |
| PUT | `/api/settings` | 更新设置 |

## 侧边栏菜单结构

```
📊 首页概览
🖥️ 虚拟机列表
📁 模板管理        ← 新增
📦 存储池          ← 新增
👤 用户管理
⚙️ 系统设置        ← 新增
```

## 默认存储路径

所有虚拟机相关文件统一存放在 `/var/lib/libvirt/` 目录下：
- 模板：`/var/lib/libvirt/images/templates`
- 虚拟机磁盘：`/var/lib/libvirt/images`
- ISO 镜像：`/var/lib/libvirt/images/ISO`

## 新增文件列表

### 后端
- `server/service/vm_create.go` - 普通创建虚拟机 service
- `server/service/storage_pool.go` - 存储池管理 service
- `server/handler/vm_create.go` - 普通创建虚拟机 handler
- `server/handler/storage_pool.go` - 存储池管理 handler
- `server/handler/settings.go` - 系统设置 handler

### 前端
- `web/src/api/infra.js` - 存储池 API
- `web/src/api/settings.js` - 系统设置 API
- `web/src/views/template/index.vue` - 模板管理页面
- `web/src/views/storage-pool/index.vue` - 存储池管理页面
- `web/src/views/settings/index.vue` - 系统设置页面

### 修改文件
- `server/model/task.go` - 新增 TaskTypeCreate 常量
- `server/router/router.go` - 新增路由
- `server/main.go` - 注册创建虚拟机任务处理器
- `web/src/api/vm.js` - 新增创建虚拟机/OS 变体 API
- `web/src/components/VmForm.vue` - 完全改造（双模式+新字段）
- `web/src/layout/index.vue` - 侧边栏新增菜单项
- `web/src/router/index.js` - 新增页面路由
