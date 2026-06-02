# 虚拟机启动冻结 CPU

## 功能说明

新增 `freeze` 配置项，用于让虚拟机在启动时先暂停 CPU，便于调试 QEMU/KVM 启动过程。

- `freeze=true`：虚拟机启动后保持暂停态，不会立即继续执行来宾系统
- `freeze=false`：保持原有行为，正常启动

## 适用范围

当前以下入口已经接入 `freeze`：

- 使用 ISO 创建虚拟机
- 从模板克隆虚拟机
- 导入已有虚拟机磁盘
- 编辑已有虚拟机的启动配置
- 面板中的虚拟机开机操作
- 由面板内部触发的虚拟机重新拉起流程，例如快照恢复、VNC 配置重载后的再次启动

以下场景暂不附带 `freeze`：

- 宿主机重启后由 libvirt 原生 `autostart` 自动拉起的虚拟机

## 前端操作

在虚拟机创建向导中，该选项位于独立的“高级设置”步骤，说明通过问号悬停提示显示。

在编辑弹窗中，该选项位于独立的“高级设置”标签页。首次进入该页面时会先显示风险提示卡片，点击“我知道了”后恢复为正常表单。

当前提示文案为：

- 虚拟机启动时自动冻结CPU（使用监视器命令c可继续启动过程）。

启用后：

- 虚拟机会以暂停态启动
- 可在 QEMU Monitor 中执行 `c`
- 也可以在面板中再次点击“开机”，此时会对暂停中的虚拟机执行恢复运行

## 开发者监视器面板

虚拟机详情页会显示“开发者功能”状态：

- 未启用时显示“未开启”
- 启用后显示“已开启”
- 鼠标悬停可查看当前已开启的开发者功能说明

虚拟机详情页右上角提供“开发者监视器”折叠入口，默认收起；点击“显示开发者页面”后，会显示对应面板，用于补齐 `qm monitor <vm>` 的常用调试入口。

更完整的说明可参考 [docs/vm-qemu-monitor.md](./vm-qemu-monitor.md)。

当前支持：

- 查看虚拟机 domain 状态与 QEMU Monitor 状态
- 所有用户都可执行 `c`
- 所有用户都可执行 `stop`
- 执行常见只读命令：`help`、`info status`、`info cpus`、`info registers`、`info block`、`info pci`、`info qtree`
- 执行当前虚拟机级别的常用操作命令：`nmi`、`system_reset`、`system_powerdown`、`system_wakeup`、`sendkey ...`
- 自定义输入受限命令：`c`、`stop`、`help`、`nmi`、`info ...`、`system_reset`、`system_powerdown`、`system_wakeup`、`sendkey ...`

当前不支持：

- 任意原始 monitor 命令透传
- 面板内直接提供类 `qm>` 的完整交互终端
- 串口控制台联动调试

典型流程如下：

1. 在创建或编辑虚拟机时启用“启动时冻结 CPU”
2. 启动虚拟机，虚拟机会进入暂停态
3. 打开详情页的“开发者监视器”标签页
4. 先执行 `info status` 确认状态
5. 准备好 VNC、日志或其他调试环境后，执行 `c` 继续启动过程

如果只是要让虚拟机继续运行，也可以直接在面板主操作区再次点击“开机”，系统会自动对暂停态虚拟机执行恢复运行。

注意：只有人为暂停或启动冻结形成的暂停态可以继续运行。如果 QEMU Monitor 显示 `paused (internal-error)`，例如 VMware 嵌套 KVM 环境中出现 `KVM: entry failed, hardware error 0x7`，则 `resume`/`c` 无法恢复，QEMU 会要求重置虚拟机。这种情况可在详情页电源管理中点击“重置”，或强制断电后重新开机；如果重置后反复出现，应优先检查宿主机嵌套虚拟化兼容性，物理机直装 KVM 通常不会遇到该问题。

## 后端实现

为避免引入数据库，`freeze` 配置通过 libvirt 域元数据保存：

- 元数据 URI：`https://kvm-console.local/domain-config`
- 元数据 key：`kvm-console`

启动虚拟机时会先读取该元数据：

- 已启用 `freeze`：执行 `virsh start <vm> --paused`
- 未启用 `freeze`：执行原有启动逻辑

## 测试记录

已在测试机 `SSH-MCP_kvm-test` 上验证：

- `virsh start test --paused` 后，`virsh domstate test --reason` 返回 `paused (user)`
- `virsh qemu-monitor-command test --hmp 'info status'` 返回 `paused (prelaunch)`
- 执行 `virsh resume test` 后可恢复为 `running (unpaused)`
