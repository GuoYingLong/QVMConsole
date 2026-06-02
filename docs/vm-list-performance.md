# 虚拟机列表性能优化

## 概述

当前虚拟机列表采用“前端缓存 + 后端数据库缓存”双层方案：

- 前端仍保留页面级缓存，减少返回列表页时的闪烁和重复 loading
- 后端新增 `vm_caches` 表，专门缓存列表、概览、下拉选择器所需的 VM 投影视图
- 管理员列表接口改为“先返回数据库缓存，再异步触发宿主机刷新”
- 普通用户列表接口始终读取数据库缓存，不再直接等待 `virsh` 全量扫描

这样可以把原先每次列表请求都依赖宿主机命令的高延迟，改成绝大多数情况下直接读 SQLite。

## 当前行为

### 1. 程序启动时自动同步 VM 缓存

- 服务启动完成数据库初始化后，会先执行一次宿主机 VM 全量同步
- 当前 libvirt 中存在的虚拟机会写入 `vm_caches`
- 若启动同步失败，会保留数据库中的旧缓存并记录中文告警，不阻断服务启动

### 2. 管理员列表接口

- `GET /api/vm/list`
- `GET /api/vm/sse`
- `GET /api/self/vms`
- `GET /api/self/vms/sse`

管理员访问以上列表类接口时：

- 立即返回数据库缓存
- 若距离上次后台刷新已超过冷却时间，则异步触发一次宿主机全量同步
- SSE 每 2 秒仍然推送一次列表快照，但不再每轮都直接全量执行 `virsh list`

### 3. 普通用户列表接口

- 普通用户所有列表类 VM 入口只读 `vm_caches`
- 过滤条件基于 `owner_username`
- 只返回 `present=true` 的可见缓存记录

当前已覆盖：

- 虚拟机列表页
- 首页 VM 概览
- 我的存储中需要选择 VM 的下拉框
- 用户分配 VM 等依赖列表接口的入口

### 4. 单台详情仍然走宿主机真实数据

以下能力仍然保持实时命令读取，不受列表缓存替代：

- 单台 VM 详情
- 单台 VM IP 查询
- 磁盘详情
- 开关机、编辑、迁移、救援等操作

这样可以保持高频列表场景提速，同时避免把实时控制逻辑完全绑定到数据库。

### 5. 写路径会主动刷新缓存

以下成功路径会立即刷新单台 VM 缓存，避免普通用户必须等待管理员刷新：

- 普通创建
- 导入 VM
- 链式克隆 / 原生链式克隆 / 批量克隆
- 轻量云开通
- 删除 VM
- 备注、开机自启、启动冻结、带宽修改
- 迁移目标节点接管 `adopt-vm`
- 用户分配 VM 后的归属同步

## 缓存字段

`vm_caches` 当前保存列表所需的核心字段：

- `name`
- `owner_username`
- `status`
- `vcpu`
- `memory_mb`
- `max_memory_mb`
- `remark`
- `template`
- `disk_size_text`
- `created_at_text`
- `autostart`
- `nic_model`
- `mac_address`
- `bandwidth_in`
- `bandwidth_out`
- `in_rescue`
- `cached_ip`
- `present`
- `last_synced_at`

其中：

- `present=false` 表示宿主机本轮同步未发现该 VM，前端默认隐藏
- 不会因为同步缺失而自动删除 VPC、轻量云配额、凭据等附属业务数据
- `owner_username` 通过现有 VM 归属文件推断，兜底为 `admin`

## 兼容说明

- 现有 `service.ListVMs` / `service.GetVM` 仍保留，宿主机真实读取逻辑没有移除
- 迁移预检、迁移执行、详情页和控制类接口继续依赖宿主机命令与现有业务表
- 普通用户“始终从数据库”当前只覆盖列表类入口，不包含详情和实时控制接口

## 涉及文件

| 文件 | 改动说明 |
|------|---------|
| `server/model/vm_cache.go` | 新增 VM 缓存表 |
| `server/service/vm_cache.go` | 新增缓存读取、宿主机同步、管理员刷新协调器 |
| `server/handler/vm.go` | 管理员列表 / SSE 切到数据库缓存 |
| `server/handler/user.go` | 普通用户列表 / SSE 改为按归属读取数据库缓存 |
| `server/main.go` | 启动时同步 VM 缓存，并在任务成功后刷新单 VM 缓存 |
| `server/service/user.go` | VM 分配后同步缓存归属 |
| `server/service/user_quota.go` | Add/Remove VM 归属时同步缓存归属 |
| `server/service/vm_config_metadata.go` | 备注修改后刷新缓存 |
| `server/service/libvirt.go` | 开机自启 / 启动冻结修改后刷新缓存 |
| `server/service/bandwidth.go` | 自定义带宽修改后刷新缓存 |
| `server/service/lightweight_cloud.go` | 轻量云配额写入后刷新缓存 |
| `server/service/vm_migration.go` | 迁移目标节点接管后立即刷新缓存 |
