# 宿主机 zRAM 配置

## 功能说明

系统设置的「宿主机设置」标签页新增 zRAM 压缩内存配置。管理员可以选择预设挡位，让宿主机创建由面板管理的 zRAM swap。

zRAM 会在内存中创建压缩块设备，不写入磁盘。对于纯虚拟化宿主机，它适合作为内存压力上升时的缓冲层：优先把可换出的页面压缩留在内存里，减少直接触发磁盘 swap 或内存不足的概率。

## 挡位说明

面板默认使用 `lz4` 压缩算法，优先控制延迟和 CPU 开销。

| 挡位 | zRAM 逻辑容量 | swap 优先级 | 适用场景 |
|------|---------------|-------------|----------|
| 关闭 | 0 | 0 | 排障，或宿主机内存压力很低 |
| 保守 | 宿主机内存 10%，最高 16 GiB | 60 | 轻量缓冲，尽量降低压缩和换页开销 |
| 均衡 | 宿主机内存 20%，最高 32 GiB | 80 | 默认推荐，适合多数纯虚拟化宿主机 |
| 积极 | 宿主机内存 35%，最高 64 GiB | 100 | VM 密度较高，希望优先压缩内存 |
| 极致 | 宿主机内存 50%，最高 128 GiB | 120 | 内存非常紧张，能接受更明显 CPU 开销 |

## 生效与持久化

切换挡位会立即执行：

1. 关闭并重置面板管理的旧 zRAM 设备。
2. 按新挡位创建 zRAM 设备。
3. 使用 `mkswap -L kvm-zram` 初始化。
4. 使用 `swapon --priority` 启用。

面板只识别和清理标签为 `kvm-zram` 的 zRAM 设备，避免误操作系统中其他 zRAM 配置。旧版本曾使用过 `kvm-console-zram` 标签，实际会被 `mkswap` 截断为 `kvm-console-zra`，当前版本会兼容识别并清理旧标签设备。

同时系统会写入：

```text
/etc/kvm-console/zram.env
/etc/systemd/system/kvm-console-zram.service
```

非关闭挡位会执行：

```bash
systemctl daemon-reload
systemctl enable kvm-console-zram.service
```

这样宿主机重启后会自动恢复面板中保存的 zRAM 挡位。

开机恢复不依赖外部脚本，`kvm-console-zram.service` 会调用面板二进制的 `host-zram-apply` 内部子命令执行恢复。

当挡位为 `off` 时，面板会禁用 `kvm-console-zram.service`，避免关闭状态仍在开机阶段调用恢复服务。

## API

### 获取 zRAM 状态

`GET /api/host/zram`

返回内容包含：

- `supported`：宿主机是否具备 zRAM 能力和 `zramctl` 工具。
- `enabled`：面板管理的 zRAM swap 是否运行。
- `current_profile`：当前运行时匹配到的挡位，若参数不属于内置挡位则为 `custom`。
- `persistent_profile`：重启后恢复的挡位。
- `runtime_config`：当前设备、容量、已用量、压缩算法、优先级等。
- `profiles`：前端可展示的挡位列表。

### 设置 zRAM 挡位

`PUT /api/host/zram`

```json
{
  "profile": "balanced"
}
```

支持的 `profile`：

- `off`
- `conservative`
- `balanced`
- `aggressive`
- `extreme`

该接口需要管理员权限，并会触发高风险二次验证。

## 注意事项

- zRAM 的容量是逻辑容量，不会一次性占满同等大小的物理内存，但不可压缩数据过多时仍然会带来实际内存压力。
- 挡位越高，压缩和换页可能越积极，CPU 开销也越明显。
- 关闭 zRAM 会先 `swapoff` 面板管理的 zRAM 设备，若设备中有大量页面，操作可能短暂增加内存压力。若 `swapoff` 已完成但内核短时间内仍标记设备 busy，面板会按关闭成功返回，并在后续切换时继续清理残留 zRAM 设备。
- 本功能不依赖数据库保存虚拟机状态，运行时信息从宿主机 `/proc`、`/sys` 和系统命令实时读取。
