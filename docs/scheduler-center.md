# 调度事件中心

## 功能说明

管理员侧边栏新增“调度事件”页面，用于统一查看系统内已注册调度器及其调度执行记录。

当前已接入以下调度器：

- `气球调度`
- `Windows 弹性内存调度`
- `虚拟机定时任务`

页面分为两部分：

- 调度器概览：显示调度器名称、键值、所属分组、启用状态、最近事件时间
- 调度事件列表：显示虚拟机、调度器类型、调度状态、调度原因、执行结果或失败原因、触发时间、完成时间

## 事件记录口径

调度事件只记录“真正发生了调度动作尝试”的场景，不记录普通轮询扫描。

例如以下情况会写入调度事件：

- 命中阈值后尝试扩容
- 命中阈值后尝试回收
- 宿主机可用内存不足导致本次调度失败
- `virsh setmem` / `virsh update-memory-device` 执行失败
- Windows 弹性内存缺少 `virtio-mem alias` 等运行时错误

以下情况不会写入调度事件：

- 未达到调度阈值
- 虚拟机未运行
- 调度器总开关关闭
- 仍处于人工暂停/冷却期

## 调度状态

页面状态固定为三种：

- `正在执行`
- `执行完毕`
- `执行失败`

后端落库流程如下：

1. 准备执行调度动作前写入 `running`
2. 调整成功后更新为 `success`
3. 调整失败后更新为 `failed`

## 数据保留

调度事件会写入数据库持久化保存，服务重启后仍可查看历史记录。

默认保留时长：

- `168` 小时（7 天）

可通过以下方式调整：

- 系统设置页中的“调度事件保留”
- 环境变量 `KVM_SCHEDULER_EVENT_RETENTION_HOURS`

后台会定时清理超出保留期的旧事件。

## 接口

### 获取调度器概览

`GET /api/scheduler/list`

返回字段：

- `key`
- `name`
- `group`
- `enabled`
- `description`
- `last_event_at`

### 获取调度事件列表

`GET /api/scheduler/events`

支持查询参数：

- `page`
- `page_size`
- `scheduler_key`
- `status`
- `vm_name`
- `start`
- `end`

返回字段：

- `id`
- `scheduler_key`
- `scheduler_name`
- `vm_name`
- `vm_backend`
- `status`
- `trigger_reason`
- `result_message`
- `error_message`
- `started_at`
- `finished_at`

### 调度事件实时推送

`GET /api/scheduler/events/sse`

SSE 事件名：

- `scheduler_event`

事件体结构：

```json
{
  "action": "upsert",
  "event": {
    "id": 12,
    "scheduler_key": "dynamic_memory_balloon",
    "scheduler_name": "气球调度",
    "vm_name": "vm-demo",
    "vm_backend": "balloon",
    "status": "success",
    "trigger_reason": "可用内存比例 10.0% 低于增长阈值 15.0%，触发扩容",
    "result_message": "已将当前内存从 2048MB 调整到 3072MB",
    "error_message": "",
    "started_at": "2026-04-27T14:00:00+08:00",
    "finished_at": "2026-04-27T14:00:02+08:00"
  }
}
```
