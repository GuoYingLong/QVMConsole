# 原生链式克隆虚拟机

## 功能概述

管理员侧“新建虚拟机”现已新增 **原生链式克隆** 模式。

该模式与普通“模板克隆”共用模板、CPU、内存、磁盘大小、网络、存储位置等表单项，但有一个关键区别：

- **模板克隆**：会继续执行模板类型对应的初始化流程，例如 Linux 的主机名/账号/密码注入、Windows 应答文件处理、FnOS 首次账号写入等。
- **原生链式克隆**：只创建 backing qcow2、定义虚拟机并直接启动，**不进入任何来宾初始化流程**。

## 适用场景

- 需要保留模板内部现有环境，不希望面板改动主机名、账号、密码或网络标识。
- 需要快速拉起测试分支机、排障机、只读验证机。
- 管理员明确知道模板内部状态，并希望自行后续处理来宾系统配置。

## 当前限制

- 仅管理员侧显示该模式。
- 仅支持从模板创建，不支持普通用户调用。
- 不会保存虚拟机登录凭据，因为该流程不会写入新凭据。
- 不会改动模板内：
  - 主机名
  - 用户名
  - 密码
  - `machine-id`
  - DHCP 租约缓存

## Windows Hyper-V 兼容

Windows 模板会自动补充 Hyper-V 来宾优化配置。由于 `stimer` 特性依赖 `hypervclock` timer，原生链式克隆在生成虚拟机 XML 时会同步确保 `<clock>` 节点中存在：

```xml
<timer name='hypervclock' present='yes'/>
```

这样可以避免 libvirt 在定义虚拟机时报错：

```text
unsupported configuration: 'stimer' hyperv feature requires 'hypervclock' timer
```

## 前端交互

位置：

- `web/src/components/VmForm.vue`

管理员在“新建虚拟机”第一步会看到额外的 **原生链式克隆** 卡片。

表单行为：

- 需要选择模板
- 需要确认磁盘大小
- 不显示登录凭据输入区
- 会优先复用模板默认硬件配置中的系统盘驱动、网卡型号和显示设备
- 仍可配置：
  - VPC / 安全组
  - 存储位置
  - 网卡类型
  - RTC
  - Guest Agent
  - SMBIOS
  - 动态内存
  - 显示设备
  - 启动固件

## 后端接口

### 提交原生链式克隆任务

`POST /api/vm/linked-clone`

仅管理员可调用，支持 API Key。

请求体示例：

```json
{
  "name": "demoalpha",
  "template": "ubuntu2404-base",
  "template_type": "linux",
  "clone_mode": "linked",
  "vcpu": 2,
  "ram": 4,
  "disk_size": 40,
  "switch_id": 3,
  "security_group_id": 2,
  "storage_pool_id": "disk-sdb1",
  "autostart": false,
  "freeze": false,
  "apic": true,
  "pae": true,
  "rtc_offset": "utc",
  "rtc_startdate": "now",
  "boot_type": "uefi",
  "nic_model": "virtio",
  "video_model": "virtio",
  "cpu_topology_mode": "auto",
  "first_boot_reboot_mode": "normal"
}
```

`clone_mode` 可选值：
- `linked`（默认）：链式克隆，创建 backing_file 链式磁盘
- `full`：完整克隆，将模板数据完整复制到独立磁盘，脱离链式条件

响应示例：

```json
{
  "code": 200,
  "message": "原生链式克隆任务已提交",
  "data": {
    "task_id": 12
  }
}
```

## 任务中心

该功能会生成新的异步任务类型：

- `linked_clone`

任务完成后返回：

```json
{
  "vm_name": "demoalpha",
  "disk_path": "/var/lib/libvirt/images/demoalpha.qcow2",
  "template": "ubuntu2404-base"
}
```

## 依赖说明

本次功能未新增系统依赖或第三方前后端库，无需更新 `docs/dependencies.md`。
