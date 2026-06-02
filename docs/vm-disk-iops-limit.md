# 虚拟机磁盘 IOPS 限制

## 功能说明

为虚拟机每个磁盘独立配置 IOPS（每秒 I/O 操作数）限制。该功能仅管理员可用，支持三种维度的限制：

- **总 IOPS**（`total_iops_sec`）：磁盘每秒总 I/O 操作数上限
- **读 IOPS**（`read_iops_sec`）：磁盘每秒读取操作数上限
- **写 IOPS**（`write_iops_sec`）：磁盘每秒写入操作数上限

所有值设为 `0` 表示不限制（默认行为）。

## 交互规则

- 仅管理员角色可见和使用此功能
- 在虚拟机**编辑表单**的磁盘列表中，每行磁盘操作列新增 `IOPS` 按钮
- 点击 `IOPS` 按钮弹出对话框，可独立设置该磁盘的总/读/写 IOPS 限制
- 设置后需点击编辑表单的「保存」按钮，IOPS 配置随编辑请求一同提交
- 在虚拟机**创建表单**（ISO 模式）中：
  - 系统磁盘区域下方有 IOPS 设置行（总/读/写三个输入框）
  - 每个额外数据盘行有 `IOPS` 按钮，点击可设置该盘的 IOPS 限制
  - 设置后创建虚拟机时会自动应用 IOPS 限制
- 系统设置中可配置**默认 IOPS 限制值**（作为参考默认值，不自动应用到已有虚拟机）

## 生效逻辑

- IOPS 限制通过 libvirt 的 `virsh blkdeviotune` 命令应用
- 虚拟机**运行中**时使用 `--live --config` 参数，限制**实时生效并持久化**
- 虚拟机**关机**时使用 `--config` 参数，确保重启后仍然生效
- IOPS 配置写入虚拟机 XML 的 `<disk><iotune>` 节点，重启后自动恢复
- 三个维度全部设为 `0` 时，清除该磁盘的所有 IOPS 限制

## 持久化机制

- IOPS 限制存储在 libvirt 的虚拟机持久化 XML 配置中
- 不依赖面板数据库存储，确保与已运行的虚拟机状态完全同步
- 虚拟机迁移、重启等操作后 IOPS 限制依然有效

## 接口说明

### 设置磁盘 IOPS 限制（仅管理员）

```
PUT /api/vm/:name/disk/:dev/iops
```

请求体：

```json
{
  "total_iops_sec": 1000,
  "read_iops_sec": 500,
  "write_iops_sec": 500
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `total_iops_sec` | int | 总 IOPS 限制，0 表示不限制 |
| `read_iops_sec` | int | 读 IOPS 限制，0 表示不限制 |
| `write_iops_sec` | int | 写 IOPS 限制，0 表示不限制 |

### 获取磁盘 IOPS 限制

```
GET /api/vm/:name/disk/:dev/iops
```

响应示例：

```json
{
  "code": 200,
  "message": "ok",
  "data": {
    "total_iops_sec": 1000,
    "read_iops_sec": 500,
    "write_iops_sec": 500
  }
}
```

### 批量设置（通过编辑 VM 接口）

管理员在 `PUT /api/vm/:name` 编辑虚拟机时，可在请求体中包含 `disk_iops` 字段：

```json
{
  "vcpu": 2,
  "memory": 2048,
  "disk_iops": {
    "vda": {
      "total_iops_sec": 1000,
      "read_iops_sec": 500,
      "write_iops_sec": 500
    }
  }
}
```

### 创建虚拟机时设置 IOPS

管理员在 `POST /api/vm/create` 创建虚拟机时，可设置：

```json
{
  "name": "test-vm",
  "vcpu": 2,
  "ram": 2048,
  "disk_size": 20,
  "system_disk_iops": {
    "total_iops_sec": 1000,
    "read_iops_sec": 500,
    "write_iops_sec": 500
  },
  "extra_disks": [
    {
      "size": 10,
      "format": "qcow2",
      "bus": "virtio",
      "iops_total": 200,
      "iops_read": 100,
      "iops_write": 100
    }
  ]
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `system_disk_iops` | object | 系统盘 IOPS 限制（仅管理员），不传或全 0 表示不限制 |
| `extra_disks[].iops_total` | int | 额外磁盘总 IOPS 限制 |
| `extra_disks[].iops_read` | int | 额外磁盘读 IOPS 限制 |
| `extra_disks[].iops_write` | int | 额外磁盘写 IOPS 限制 |

### 磁盘列表返回 IOPS 信息

`GET /api/vm/:name/disks` 返回的每个 `DiskInfo` 新增三个 IOPS 字段：

```json
{
  "device": "vda",
  "iops_total": { "value": 1000, "is_set": true },
  "iops_read": { "value": 500, "is_set": true },
  "iops_write": { "value": 500, "is_set": true }
}
```

## 系统设置

系统设置（`/api/settings`）中新增三个默认 IOPS 配置项：

| 设置项 | 环境变量 | 默认值 |
|--------|----------|--------|
| `default_disk_iops_total` | `KVM_DEFAULT_DISK_IOPS_TOTAL` | `0` |
| `default_disk_iops_read` | `KVM_DEFAULT_DISK_IOPS_READ` | `0` |
| `default_disk_iops_write` | `KVM_DEFAULT_DISK_IOPS_WRITE` | `0` |

这些默认值仅作为创建虚拟机时的**参考默认值**，不会自动应用到已有的虚拟机磁盘。现有虚拟机需要在编辑页面中单独配置。

## 实现位置

- 后端 IOPS 操作：`server/service/disk.go` - `SetDiskIOPSTune`, `GetDiskIOPSTune`, `ParseAllDiskIOPSTune`
- 后端 IOPS 端点：`server/handler/disk.go` - `SetDiskIOPS`, `GetDiskIOPS`
- 后端编辑集成：`server/handler/vm.go` - `VmEditRequest.DiskIOPS`
- 后端系统设置：`server/handler/settings.go` - `SettingsResponse.DefaultDiskIOPSTotal` 等
- 后端配置模型：`server/config/config.go` - `Config.DefaultDiskIOPSTotal` 等
- 后端路由注册：`server/router/router.go`
- 前端 API：`web/src/api/vm.js` - `getDiskIOPS`, `setDiskIOPS`
- 前端编辑表单：`web/src/components/VmForm.vue` - IOPS 按钮 + 对话框
- 前端详情页：`web/src/views/vm/detail.vue` - 磁盘 IOPS 信息卡片
- 前端系统设置：`web/src/views/settings/index.vue` - 默认 IOPS 配置
