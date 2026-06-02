# Windows 模板克隆系统盘自动扩容

## 功能说明

Windows 模板克隆时，如果用户填写的系统盘容量大于模板原始容量，后端会自动处理系统盘扩容。

## Windows 版本分类与启动适配

Windows 模板现在支持二级分类：`WindowsServer2022`、`Windows10`、`WindowsServer2012R2`。分类只影响模板管理和创建入口展示，克隆逻辑仍以模板 `.meta.json` 中的 `boot_type` 和 `default_config` 为准。

`WindowsServer2012R2.qcow2` 建议保留模板元数据中的 `boot_type: bios`，并在默认配置中使用模板实际可启动的磁盘、网卡和显示设备，例如 `disk_bus: sata`、`nic_model: e1000e`、`video_model: vmvga`。从模板创建时，后端会按这些配置生成 BIOS/SATA 等兼容 XML，不再强制套用 Windows Server 2022 的 UEFI Secure Boot 和 VirtIO 系统盘路径。

如果 2012 R2 等旧模板使用 MBR/msdos 分区表，后端只检测基础分区信息并跳过 GPT 专属元数据读取，避免 `part_get_gpt_guid` 在非 GPT 磁盘上报错。若最大的 NTFS 系统分区已经是最后一个分区，后端会直接把该分区扩到磁盘末尾并执行 `ntfsresize`；若系统分区后还有其他分区，则不会移动分区，克隆任务会继续创建虚拟机，剩余空间需要进入 Windows 后手动处理。

当前流程分为两步：

1. 宿主机离线调整 GPT 分区表。
   - GPT 模板如果系统分区后方存在 Windows 恢复分区，会先使用 `ntfsclone` 备份恢复分区，再把恢复分区移动到新磁盘末尾。
   - MBR/msdos 模板如果系统分区已经是最后一个分区，会直接扩展该分区，不执行 GPT 专属元数据读取。
   - 系统分区边界会扩展到恢复分区之前，保留恢复分区本身。
   - 分区表调整完成后会立即执行 `ntfsresize -f` 扩展 NTFS 文件系统。
   - 任务进度会把恢复分区移动和 NTFS 扩展拆开显示，方便判断耗时阶段。
   - 如果 NTFS 文件系统不一致导致扩容失败，克隆任务会报错终止。此时需要先启动模板执行 `chkdsk /f`，正常关机后重新制作模板。

2. Windows 首次启动不再执行额外扩容脚本。
   - 扩容已经在宿主机离线阶段完成。
   - 启动后不会弹出扩容命令窗口，避免用户看到无意义的红色脚本错误。
   - 如果离线扩容失败，克隆任务会直接失败并显示错误原因。

## 适用布局

已在测试机 `wintest` 验证过的 ISO 安装布局：

| 分区 | 类型 |
|------|------|
| `sda1` | EFI FAT32 |
| `sda2` | MSR |
| `sda3` | Windows NTFS 系统分区 |
| `sda4` | Windows Recovery NTFS 恢复分区 |

该布局下，克隆 20G 模板为 30G 磁盘后，后端会把恢复分区移动到磁盘末尾，并把系统分区边界扩展到约 29G。

## 依赖

使用的宿主机命令均来自已有依赖 `libguestfs-tools`：

- `guestfish`
- `virt-filesystems`
- `virt-customize`
- `virt-win-reg`
- `ntfsclone`
- `ntfsfix`

详细依赖清单见 `docs/dependencies.md`。

## 注意事项

- Windows 文件系统实际可用容量在克隆任务完成前已经扩展完成。
- 如果分区布局不是 GPT，或系统分区后方存在非恢复分区，后端会跳过离线移动；此类特殊布局需要进入 Windows 后手动处理。
- 如果任务停留在“移动 Windows 恢复分区并扩展系统分区”，后端会在外部命令超时或任务取消时清理整组 `guestfish/qemu/ntfsresize` 子进程，避免残留进程继续占用已删除的克隆磁盘。
- 超时通常说明模板不是正常关机状态、NTFS 文件系统存在错误，或宿主机磁盘 IO 过慢。建议先启动模板执行 `chkdsk /f`，正常关机后重新制作模板再克隆。
