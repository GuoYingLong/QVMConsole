# 模板默认硬件配置

## 功能说明

制作模板时，系统现在会自动记录源虚拟机的关键硬件信息，并在下次“从模板创建”或“原生链式克隆”时作为前端默认值自动选中，减少重复填写和误选。

当前记录字段：

- CPU 核心数
- 内存大小（GB）
- 系统盘大小（GB）
- 系统盘总线类型（`virtio` / `scsi` / `sata` / `ide`）
- 首张网卡型号（`virtio` / `e1000e` / `rtl8139`）
- 显示设备型号（`virtio` / `vga` / `vmvga` / `cirrus`）
- CPU 拓扑模式（`auto` / `single_socket` / `host_default`）
- Windows 首次重启策略（`normal` / `cold`）

## 使用方式

1. 在虚拟机列表中选择一台已关机的虚拟机，点击“制作模板”。
2. 系统会在复制模板磁盘时自动采集该虚拟机的 CPU、内存、系统盘大小、系统盘总线、网卡型号、显示设备和 CPU 拓扑模式。
3. 之后在新建虚拟机向导中选择该模板：
   - CPU、内存会自动带出到“硬件规格”步骤；
   - 系统盘大小和系统盘驱动会自动带出到“基础信息”步骤；
   - 网卡型号会自动带出到“网络设置”步骤；
   - 显示设备、CPU 拓扑和首次重启策略会自动带出到“高级设置”步骤；
   - 引导固件仍继续优先复用模板元数据中的 `boot_type`。
   - 前端在选择模板时会主动刷新一次最新模板元数据，确保刚修改过的默认值也能立即带出。
4. 如果需要调整已有模板或旧模板的默认值，可在“模板管理 → 设置”中手动编辑这些字段并保存。

## 兼容策略

- 旧模板没有这批字段时不会报错，也不需要重做模板。
- 旧模板可以直接在“模板管理 → 设置”中补录这些字段；首次保存时系统会自动创建 `.meta.json`。
- 旧模板仍会继续沿用当前默认行为：
  - 系统盘大小继续按模板磁盘最小值回填；
  - 系统盘总线和网卡型号回退到现有默认值；
  - CPU 和内存保持表单原始默认值，管理员可手动调整。

## 接口与元数据

模板 `.meta.json` 新增 `default_config` 对象，例如：

```json
{
  "default_config": {
    "vcpu": 4,
    "ram": 8,
    "disk_size": 80,
    "disk_bus": "sata",
    "nic_model": "e1000e",
    "video_model": "vmvga",
    "cpu_topology_mode": "auto",
    "first_boot_reboot_mode": "cold"
  }
}
```

说明：

- `POST /api/template/prepare` 无需新增请求参数，字段由后端自动采集。
- `GET /api/template/list` 会返回 `default_config`，供前端默认值回填使用。
- `PUT /api/template/:name/publish` 现支持同时更新 `default_config` 对应字段，供管理员手动维护已有模板默认值。
- `POST /api/vm/clone` 与 `POST /api/vm/linked-clone` 现支持可选 `disk_bus`、`video_model`、`cpu_topology_mode` 和 `first_boot_reboot_mode`；未显式传入时，后端会优先复用模板记录的磁盘总线类型、显示设备型号、CPU 拓扑模式和首次重启策略。

## Windows 热重启兼容说明

部分 Windows 模板在 VMware 嵌套虚拟化环境中，来宾系统首次初始化后自动重启可能出现 VNC 黑屏或提示 `Guest has not initialized the display (yet)`。这类场景可能与显示设备模型、CPU 拓扑或 QEMU 软复位链路有关：如果源虚拟机使用 `vmvga` 或其他兼容显卡，克隆后应继续沿用同一显示设备；Windows 10/11 客户端建议使用 `auto` 或 `single_socket`，避免将 6 核暴露为 6 个物理 CPU 插槽。

如果确认模板在 Windows 自己触发的首次重启后黑屏，但关机后重新开机可以正常进入系统，可将 `first_boot_reboot_mode` 设置为 `cold`。此模式只影响 Windows 模板克隆的首次启动阶段：面板会临时让首次来宾重启转换为关机，然后从宿主机重新开机，完成后恢复后续正常重启策略。

已有模板如果缺少 `default_config.video_model` 或 `default_config.cpu_topology_mode`，后端会优先尝试从元数据中的 `created_from_vm` 来源虚拟机读取；如果来源虚拟机已不存在，可在“模板管理 → 发布设置 → 默认创建配置”中手动补录。修改后，新建的模板克隆和原生链式克隆会自动带出该值；已创建的虚拟机需要关机后在虚拟机编辑页修改显示设备或 CPU 拓扑。
