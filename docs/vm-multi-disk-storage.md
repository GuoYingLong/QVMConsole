# 虚拟机多硬盘与迁移存储选择

## 功能概述

创建、编辑和跨节点迁移虚拟机时，硬盘存储选择按以下规则处理：

- 新建虚拟机的动态内存开启后会展示内存调度方式，可选择气球调度；Windows 目标可选择 Windows 弹性内存。
- 普通 ISO 创建和模板创建都支持额外添加多块硬盘，额外硬盘会随创建任务自动挂载。
- 新建虚拟机的额外硬盘支持单独选择存储位置；留空时使用上方虚拟机硬盘默认存储位置。
- 普通用户自助创建时，系统盘和额外硬盘容量会合并计入硬盘配额，超过配额会拒绝提交。
- 编辑虚拟机新增硬盘时，可以为每块新增硬盘选择存储位置；留空则使用默认虚拟机存储位置。
- 跨节点迁移多硬盘虚拟机时，可以为每块硬盘选择目标节点上的存储位置，预检会按存储分别检查可用空间、目标路径冲突和 backing 链。

## 接口变更

### 普通用户自助创建

`POST /api/self/vm/create` 新增兼容字段：

```json
{
  "extra_disks": [
    {
      "size": 20,
      "format": "qcow2",
      "bus": "virtio"
    }
  ]
}
```

普通用户额外硬盘格式会按 `qcow2` 处理，容量会和系统盘一起参与配额校验。`storage_pool_id` 留空时使用默认虚拟机存储位置。

### 模板克隆创建

`POST /api/vm/clone`、`POST /api/self/vm/clone` 和 `POST /api/vm/linked-clone` 支持同样的 `extra_disks` 字段：

```json
{
  "template": "ubuntu-template",
  "name": "demo",
  "extra_disks": [
    {
      "size": 50,
      "format": "qcow2",
      "bus": "virtio",
      "storage_pool_id": "sdb1"
    }
  ]
}
```

普通用户从模板创建时，模板系统盘扩容后的容量和额外硬盘容量会合并计入硬盘配额。

### 编辑虚拟机新增硬盘

`PUT /api/vm/:name` 的 `add_disks` 支持 `storage_pool_id`：

```json
{
  "add_disks": [
    {
      "size": 20,
      "format": "qcow2",
      "bus": "virtio",
      "storage_pool_id": "sdb1"
    }
  ]
}
```

### 跨节点迁移

`POST /api/vm/:name/migration/preview` 和 `POST /api/vm/:name/migrate` 新增 `disk_storage_targets`：

```json
{
  "target_storage_pool_id": "sda1",
  "disk_storage_targets": [
    {
      "target": "vda",
      "target_storage_pool_id": "sda1"
    },
    {
      "target": "vdb",
      "target_storage_pool_id": "sdb1"
    }
  ]
}
```

`target_storage_pool_id` 仍作为默认目标存储，未单独指定的硬盘会使用该默认值。多硬盘迁移建议为每块硬盘显式选择目标存储，避免大盘全部落到同一块目标硬盘。

## 兼容性

旧客户端只传 `target_storage_pool_id` 时行为不变，所有硬盘仍迁移到同一个目标存储。新字段只影响显式指定的硬盘。

本功能不新增宿主机依赖。
