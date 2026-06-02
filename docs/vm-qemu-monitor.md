# 虚拟机开发者监视器

## 功能说明

虚拟机详情页新增“开发者监视器”折叠入口，用于补齐常见的 QEMU Monitor 调试操作，方便配合启动冻结功能观察虚拟机早期启动过程。

当前面板对有虚拟机访问权限的用户开放，所有开放命令都只作用于当前虚拟机本身，不提供宿主机暴露、磁盘链修改、迁移、快照底层操作等高风险能力。

## 适用场景

适合以下调试场景：

- 虚拟机启用“启动时冻结 CPU”后，需要手动继续启动
- 启动前希望先打开 VNC、日志、串口或其他调试环境
- 需要查看 QEMU 当前运行状态
- 需要执行常见只读 monitor 命令排查设备或 CPU 状态

该面板不等同于 VNC，也不等同于串口控制台：

- VNC 负责显示图形输出
- QEMU Monitor 负责控制虚拟机执行状态
- 串口控制台负责观察来宾系统串口输出

## 当前支持

面板提供以下能力：

- 查看虚拟机当前 domain 状态
- 查看 QEMU Monitor 返回的状态
- 快捷执行 `c`
- 快捷执行 `stop`
- 快捷执行 `help`
- 快捷执行 `nmi`
- 快捷执行 `info status`
- 快捷执行 `info cpus`
- 快捷执行 `info registers`
- 快捷执行 `info block`
- 快捷执行 `info pci`
- 快捷执行 `info qtree`
- 快捷执行 `system_reset`
- 快捷执行 `system_powerdown`
- 快捷执行 `system_wakeup`
- 快捷执行 `sendkey ctrl-alt-delete`

自定义输入框目前也支持，但仅允许以下命令范围：

- `c`
- `stop`
- `help`
- `nmi`
- `info ...`
- `system_reset`
- `system_powerdown`
- `system_wakeup`
- `sendkey ...`

## 当前限制

当前版本暂不支持：

- 任意原始 QEMU Monitor 命令透传
- 类似 `qm monitor` 的完整交互终端
- 直接在面板里接入串口控制台
- 会改宿主机网络、磁盘链、迁移状态、快照链或调试暴露面的命令

这样做是为了先满足常用调试流程，同时避免把高风险 monitor 命令直接暴露到面板。

## 典型使用流程

1. 在虚拟机创建向导或编辑弹窗中启用“启动时冻结 CPU”
2. 启动虚拟机
3. 打开虚拟机详情页
4. 点击右上角“显示开发者页面”
5. 先执行 `info status` 确认虚拟机是否处于 `paused (prelaunch)` 或其他暂停态
6. 准备好调试环境后执行 `c`
7. 若需要再次暂停，可执行 `stop`

首次打开“开发者监视器”时，面板会先显示一张说明卡片，并重点提示：

- 若您不是开发者或专业人士，请不要使用此功能，避免影响业务

## 与启动冻结功能的关系

如果虚拟机启用了“启动时冻结 CPU”，启动后通常会表现为：

- VNC 可连接，但画面可能显示 `Display output is not active.`
- 虚拟机状态为 `paused`
- 需要执行 `c` 后才会继续启动来宾系统

如果只是想让虚拟机继续运行，也可以直接在详情页主操作区再次点击“开机”。对于暂停态虚拟机，面板后端会自动执行恢复运行。

如果 `info status` 返回 `paused (internal-error)`，这不是启动冻结或手动 `stop` 形成的可恢复暂停态。常见场景是在 VMware 等嵌套虚拟化环境中，QEMU 日志出现 `KVM: entry failed, hardware error 0x7`；此时执行 `c` 或面板“继续启动”会失败，并提示需要重置虚拟机。处理方式是在详情页电源管理中点击“重置”，或强制断电后重新开机；若每次启动都复现，应检查上层虚拟化平台是否完整开放嵌套 VT-x/AMD-V，或改用物理机运行。

## 测试记录

已在测试机 `SSH-MCP_kvm-test` 上验证：

- `virsh start test --paused` 后，`virsh domstate test --reason` 返回 `paused (user)`
- `virsh qemu-monitor-command test --hmp 'info status'` 返回 `VM status: paused (prelaunch)`
- 执行 `virsh qemu-monitor-command test --hmp 'c'` 后可恢复为 `running (unpaused)`
- 执行 `virsh qemu-monitor-command test --hmp 'stop'` 后可再次回到 `paused`
- `help nmi`、`help sendkey`、`help system_wakeup` 均可正常返回帮助信息
