# 宿主机 KSM 配置

## 功能说明

系统设置新增「宿主机设置」标签页，管理员可以在这里配置 KSM（Kernel Samepage Merging）内存去重。

KSM 是宿主机级能力，会扫描匿名内存页，并把内容完全相同的页面合并为同一份物理内存。对于纯虚拟化宿主机，如果大量虚拟机来自相似模板，通常能降低宿主机内存占用。

## 挡位说明

| 挡位 | 参数倾向 | 适用场景 |
|------|----------|----------|
| 关闭 | `run=0` | 临时排障，或宿主机 CPU 压力明显高于内存压力 |
| 保守 | 低频扫描，不跨 NUMA 节点 | 内存压力不高，希望尽量降低 CPU 开销 |
| 均衡 | 中等扫描频率，跨 NUMA 节点，启用零页合并 | 默认推荐，适合大多数纯虚拟化宿主机 |
| 积极 | 高频扫描，启用零页合并 | VM 密度较高，希望更快释放重复内存 |
| 极致 | 最高扫描频率，启用零页合并 | 内存非常紧张，能接受更明显 CPU 开销 |

当前后端写入的核心参数包括：

- `/sys/kernel/mm/ksm/run`
- `/sys/kernel/mm/ksm/pages_to_scan`
- `/sys/kernel/mm/ksm/sleep_millisecs`
- `/sys/kernel/mm/ksm/merge_across_nodes`
- `/sys/kernel/mm/ksm/use_zero_pages`
- `/sys/kernel/mm/ksm/smart_scan`（内核支持时写入）

## 生效与持久化

切换挡位后会立即写入宿主机 sysfs，因此当前运行中的虚拟机也会受到 KSM 扫描策略影响。

同时系统会写入：

```text
/etc/kvm-console/ksm.env
/etc/systemd/system/kvm-console-ksm.service
```

并执行：

```bash
systemctl daemon-reload
systemctl enable kvm-console-ksm.service
```

这样宿主机重启后会自动恢复面板中保存的 KSM 挡位。

## API

### 获取 KSM 状态

`GET /api/host/ksm`

返回内容包含：

- `supported`：宿主机是否提供 KSM sysfs 接口
- `enabled`：当前 KSM 是否运行
- `current_profile`：当前运行时匹配到的挡位，若参数不属于内置挡位则为 `custom`
- `persistent_profile`：重启后恢复的挡位
- `runtime_config`：当前 sysfs 参数
- `metrics`：KSM 共享页、扫描页等统计
- `profiles`：前端可展示的挡位列表

### 设置 KSM 挡位

`PUT /api/host/ksm`

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

- KSM 会带来后台扫描 CPU 开销，挡位越高越明显。
- 关闭 KSM 不会立刻拆开所有已经合并的页面，内核会在页面被写入时逐步拆分。
- 如果宿主机未提供 `/sys/kernel/mm/ksm/run`，页面会显示不支持，设置接口会拒绝写入。
- 本功能不依赖数据库保存虚拟机状态，所有运行时信息均从宿主机 sysfs 实时读取。
