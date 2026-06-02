# UEFI 内存快照 NVRAM 兼容说明

## 背景

UEFI 虚拟机使用 pflash 固件时，libvirt 创建内部快照会同时处理 NVRAM 变量文件。若 NVRAM 是 `raw` 格式，创建包含内存状态的内部快照会失败，并提示：

```text
internal snapshots of a VM with pflash based firmware require QCOW2 nvram format
```

## 当前处理

- 新建、导入、模板克隆和原生链式克隆的 UEFI 虚拟机会把 NVRAM 生成为 `qcow2` 格式。
- 从已有 UEFI 模板克隆时，会把模板 NVRAM 转换为 `qcow2` 后再写入新 VM。
- 已存在的老 VM 如果仍是 `raw` NVRAM：
  - 虚拟机关机时创建内部快照，会自动把 NVRAM 转换为 `qcow2`，并保留原文件备份。
  - 虚拟机运行中创建内存快照时，第一次提交会返回需要修复的提示，页面会弹出二次确认。
  - 用户确认后，任务队列会正常关机、转换 NVRAM、重新开机，然后继续创建内存快照。

## 操作建议

运行中的老 UEFI VM 第一次遇到该问题时，建议在业务低峰期确认自动修复。自动修复只发送正常关机指令；如果虚拟机 180 秒内没有关机，系统不会强制断电，会提示用户先在系统内关机后再重试。

若 VM 已经运行并且只是需要一个不包含内存状态的快照，可以不勾选“创建快照时保存虚拟机内存状态”，系统会继续使用外部磁盘快照。

API 调用方如果要确认自动修复，可在创建快照请求中传入 `auto_fix_nvram: true`。未传该字段时，后端会以 `409` 返回 `require_nvram_fix: true`，用于前端弹出二次确认。

## 依赖

该兼容处理未新增 apt 依赖，复用已有 `qemu-img`、`virsh` 和 OVMF 固件文件。
