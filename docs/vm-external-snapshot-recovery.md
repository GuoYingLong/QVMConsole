# 外部快照恢复合并说明

## 背景

运行中的虚拟机创建“仅磁盘”快照时，libvirt 会生成外部 qcow2 overlay。恢复外部快照时，面板会以快照创建前的磁盘作为 backing，新建一层 `*.snap_restore_*.qcow2` 可写 overlay，并把虚拟机切到这层 overlay 上运行；恢复流程不会执行 `qemu-img commit`，也不会自动删除快照元数据或原快照 overlay 文件，避免把快照后的写入合并回去或让快照列表被清空。删除外部快照时，才会按删除操作清理对应元数据和 overlay 文件。

## 合并范围

外部快照 XML 中同时包含：

- 顶层 `<disks>`：本次快照实际创建的 overlay 文件。
- `<domain>` 和 `<inactiveDomain>`：创建快照时虚拟机的完整磁盘配置，里面可能包含历史 overlay 和 backing 链。

恢复和删除外部快照时，只应识别顶层 `<disks>` 中记录的 overlay 文件，不能把 `<domain>` 里的历史 backing 链当成本次快照文件。恢复外部快照时不合并这些 overlay，只把快照创建时的磁盘路径作为新恢复 overlay 的 backing；删除外部快照时才按删除语义处理 overlay。

## 多磁盘恢复

恢复外部快照时，需要把当前 VM 磁盘切回快照 XML 的 `<domain>` 中记录的创建快照前磁盘。多磁盘 VM 必须按 `target dev`（如 `vda`、`vdb`）匹配原始磁盘路径，不能按 XML 中所有 `<source file>` 的出现顺序匹配，因为 backingStore 也会包含 `<source file>`，顺序匹配会把某块盘切到另一块盘的 backing 层。

恢复完成后还需要执行 `virsh snapshot-current <vm> <snap>` 同步 libvirt 当前快照标记。否则从内部快照恢复到外部快照时，磁盘可能已经切回外部快照点，但快照树仍会显示父级内部快照为当前，后续创建快照也可能挂到错误父节点。

恢复外部快照后不能让 VM 直接写入快照创建前的磁盘文件。否则从最早快照恢复后再创建新的分支快照，后续写入会落进最早快照的 backing，导致最早快照被新分支数据污染。面板因此会在恢复时自动创建 `*.snap_restore_*.qcow2` 分支 overlay；后续创建的新快照会基于这层分支 overlay 继续生成，旧恢复点保持不变。

如果外部快照或数据盘位于 `/var/lib/kvm-storage` 自定义宿主机存储池，恢复后启动前会主动修复相关磁盘文件的 `libvirt-qemu:kvm` 权限，并确保 libvirt / AppArmor 允许 `virt-aa-helper` 和 QEMU 访问该路径。这样可以避免快照 overlay 文件没有 `.qcow2` 后缀时被 AppArmor 拒绝读取，导致启动时报 `Permission denied`。

内部快照恢复前也会执行同一套权限修复。面板会读取当前磁盘和快照 XML 中记录的磁盘路径，并通过 `qemu-img info --backing-chain` 顺着 qcow2 backing chain 找到模板盘，例如 `/var/lib/libvirt/images/templates/*.qcow2`，一并修复为 QEMU 可读。这样可以避免链式克隆 VM 恢复内部快照时，当前盘权限正常但底层模板盘被 QEMU 报 `Permission denied`。

## 删除顺序

快照如果仍有子快照，不能直接删除父级节点。内部快照如果已经成为外部快照链的父节点，且当前 VM 正在使用外部 overlay，libvirt 也不能直接删除这个父级内部快照。后端会在删除前检查 `Children`，发现仍有子快照时直接返回中文提示，避免执行会破坏快照树或只清理元数据的危险操作；页面也会在提交任务前拦截这类删除。

这类快照应按快照树从叶子节点开始处理；如果确实要删除父级内部快照，需要先恢复/切回该内部快照，确认当前活动磁盘已经回到内部快照所在的原始磁盘后再删除。

如果叶子外部快照记录已被删除，但 VM 仍然运行在面板生成的 `.snap_...` 外部 overlay 上，后端删除内部快照时会先尝试折叠当前活动 overlay，然后重试删除内部快照。若 overlay 的 backing 仍是其他外部快照的恢复点，不能执行 `blockcommit` 污染 backing；运行中 VM 会改用 `virsh blockcopy --pivot` 复制成独立当前盘。这样更接近 VMware 的“删除快照但保留当前状态”语义；非面板生成或无法识别的外部链不会自动合并，避免误把模板链式克隆合并到模板盘。

单独删除外部快照时也遵循同样规则：非活动 overlay 不再 commit 回 backing；当前活动 overlay 只有在不会影响其他仍存在的外部快照恢复点时才使用 `blockcommit`，否则使用 `blockcopy` 独立当前盘。

页面和 API 提供“删除全部快照”能力，后端会循环选择快照树叶子节点删除，直到快照清空。删除虚拟机前也会调用同一套清空快照逻辑，确保转移或删除磁盘前先折叠面板生成的外部 overlay。

如果“删除全部快照”遇到历史内部快照所在磁盘与当前活动磁盘不一致，且当前活动链里已经没有可自动合并的面板外部 overlay，后端会仅删除该快照的 libvirt 元数据，继续清空快照树。这表示面板不再把该旧恢复点视为可恢复快照；不会强行把未知旧磁盘文件切回当前 VM 或执行 `qemu-img commit`，避免为了清理快照列表再次污染当前磁盘状态。单独删除内部快照仍保持严格保护，遇到同类错位会返回错误并要求人工确认磁盘链。

如果清空任务已经尝试折叠当前外部 overlay，但 libvirt 重试删除内部快照时仍然报告磁盘不一致，也会按删除全部语义降级为仅清理该内部快照元数据。这通常表示旧内部快照属于已经分叉或切换过的历史磁盘文件；继续强行切盘或合并反而可能破坏当前 VM 状态。

单独删除快照和删除全部快照完成后，后端会继续清理面板生成的残留文件。清理范围只包括当前 VM 名称下的 `.snap_*`、`.snap_restore_*` 等快照/恢复 overlay 文件；删除前会保护当前 VM 磁盘、当前 backing chain、QEMU 当前块设备，以及剩余快照 XML 中仍引用的文件。删除全部快照还会先尝试折叠当前活动的面板快照 overlay，确保最终不会留下仍可安全清理的快照痕迹。

## 超时策略

`qemu-img commit` 属于磁盘级耗时操作，overlay 较大或宿主机 I/O 较慢时可能超过普通命令的 30 秒超时。当前仅删除外部快照时使用长耗时命令执行路径；恢复外部快照不会执行 commit。

## 生产排查建议

如果外部快照恢复失败，请先只读确认当前状态：

```bash
virsh domstate <vm_name> --reason
virsh domblklist <vm_name> --details
virsh snapshot-list <vm_name> --tree
qemu-img info --backing-chain <current_overlay>
```

确认文件仍在 backing chain 中时，不要直接删除 overlay。若 `qemu-img check` 只提示 leaked clusters，通常表示空间泄漏而非数据损坏，但修复仍属于写操作，建议在虚拟机关机、链条和备份方案确认后再执行。
