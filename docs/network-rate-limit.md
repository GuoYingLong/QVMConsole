# 虚拟机网卡速率限制

## 功能概述

为虚拟机网卡提供外网带宽速率限制功能。运行中的 VM 使用 OVS `linux-htb` QoS 队列限制下行外网流量，使用 OpenFlow meter 限制上传外网流量；内网网段流量不限制；`virsh domiftune --config` 仅用于持久化和读取面板中的限速配置。

## 核心概念

| 参数 | 含义 | 单位 | 控制方 |
|------|------|------|--------|
| **平峰速率 (average)** | 持续带宽上限 | Mbps | 系统自动均分 / 用户可编辑 |
| **峰值速率 (peak)** | 最大瞬时速率 | Mbps | = 用户配额，不可修改 |
| **突发量 (burst)** | 允许突发的总数据量 | KB | = 系统最大速率 × 30秒，不可修改 |

### 限速机制

1. VM 运行时，后端根据静态 DHCP 绑定或租约获取 VM 的内网 IP
2. OVS 流表优先放行 `192.168.122.0/24` 内网流量，VM 与宿主机、VM 与 VM 的内网通信不走限速
3. VM 访问内网网段之外的地址时，通过 OVS meter 限制上传流量
4. 外部回程流量进入 VM 时，通过 OVS `set_queue` 进入下行限速队列
5. 关机 VM 的配置仍写入 libvirt config，VM 下次启动后重新应用 OVS QoS/Queue/meter/flow
6. VM 每次重新启动、重装系统后重新开机，后端都会按持久化配置重刷运行态带宽规则，避免 `vnet`/`ofport` 变化后旧 OVS 精确流仍指向历史端口，导致外网回程流量黑洞

### 示例

**场景一：仅全局限速**

- 系统设置下行50Mbps、上行50Mbps
- 有效带宽：45Mbps下行、45Mbps上行
- 当前运行3台VM

每台 VM 的限速参数：
- 下行平峰速率 = **45Mbps**（每台VM均设置全量有效带宽）
- 上行平峰速率 = **45Mbps**
- 峰值速率 = **45Mbps**（与平峰一致）
- 突发量 = 45 × 125 × 30 = **168750KB**
- 当多台VM同时跑满时，由TCP拥塞控制在各VM之间自然分享带宽

**场景二：用户配额 + 全局限速**

- 系统设置下行100Mbps（有效95Mbps）
- 用户下行配额：40Mbps
- 用户拥有 2 台 VM

每台 VM 的限速参数（取两者中更严格的）：
- 全局限速：每台 VM 95Mbps（全量有效带宽）
- 用户配额均分：40/2 = 20Mbps
- 实际生效：min(95, 20) = **20Mbps**（用户配额更严格）

## 使用方法

### 1. 系统设置（全局带宽限制）

进入 **系统设置 → 存储与网络 → 全局带宽限制**：

- **下行总带宽(Mbps)**：全局限速下行总带宽，配置后有效带宽 = 配置值 - 5Mbps（保留5Mbps缓冲），所有运行中VM平分，0=不限制
- **上行总带宽(Mbps)**：全局限速上行总带宽，配置后有效带宽 = 配置值 - 5Mbps（保留5Mbps缓冲），所有运行中VM平分，0=不限制

> **核心规则**：
> - 当管理员配置上下行带宽后，所有非轻量云虚拟机的外网带宽被限制为 `(配置值-5) / 运行中VM数量` Mbps
> - 例如：配置下行50Mbps，10台VM运行，每台限速4.5Mbps下行
> - 配置为0时清除所有全局限速，恢复由用户配额控制的带宽分配
> - VM开机/关机/强制断电后自动重新均分
> - 轻量云VM不受全局限速影响（由单机配额管理）
> - VPC交换机内的VM由交换机网关聚合限速

环境变量：`KVM_MAX_BURST_INBOUND` / `KVM_MAX_BURST_OUTBOUND`

### 2. 用户配额

进入 **用户管理 → 编辑配额**：

- **下行带宽**：用户所有 VM 的下行带宽总配额（Mbps），0=不限制
- **上行带宽**：用户所有 VM 的上行带宽总配额（Mbps），0=不限制

保存配额后，系统自动将带宽均分到用户所有 VM。

### 3. VM 网络管理

进入 **虚拟机详情 → 网络管理 → 速率限制**，或在虚拟机列表中打开 **网络管理 → 速率限制**：

- **下行速率**：该 VM 的下行平峰速率（Mbps）
- **上行速率**：该 VM 的上行平峰速率（Mbps）

> 注意：普通用户修改时，后端会校验单台 VM 和所有 VM 的平峰速率总和不超过用户配额。若用户某个方向存在有限配额，该方向不能设置为 0，避免绕过配额。

### 4. 自动重分配

以下操作会触发自动重新均分所有 VM 的平峰速率：

**用户配额重分配：**
- 管理员更新用户配额
- 用户创建/克隆新 VM
- 用户删除 VM
- 管理员为用户分配 VM
- VM 开机时

**全局带宽重分配（管理员配置了全局限速时）：**
- 管理员保存系统带宽设置
- 任何 VM 开机/关机/强制断电
- 全局限速在VM启动/关闭后自动重新均分给所有运行中VM

> 重分配会覆盖用户之前的自定义平峰速率设置。

## API 变更

### 系统设置

```
GET  /api/settings       → 新增 max_burst_inbound, max_burst_outbound
PUT  /api/settings       → 支持设置 max_burst_inbound, max_burst_outbound
```

### 用户管理

```
POST /api/user                → 新增 max_bandwidth_up, max_bandwidth_down
PUT  /api/user/:username/quota → 新增 max_bandwidth_up, max_bandwidth_down（保存后自动重分配）
GET  /api/user                → 返回带宽配额信息
```

### 虚拟机

```
PUT  /api/vm/:name  → 新增 bandwidth_inbound_avg, bandwidth_outbound_avg
GET  /api/vm/:name  → 返回 bandwidth_in, bandwidth_out 和 bandwidth 详情
GET  /api/vm        → 列表返回 bandwidth_in, bandwidth_out
```

## 技术实现

运行态限速使用 OVS QoS/Queue、OpenFlow meter 和 `ovs-ofctl -O OpenFlow13` 流表：

- VM 上行外网：匹配 VM vnet 端口、`nw_src=<VM IP>`，目的地址不属于内网放行规则时经过 OVS meter 后输出到 `LOCAL`
- VM 下行外网：匹配 `LOCAL` 进入 OVS、`nw_dst=<VM IP>`，来源地址不属于内网放行规则时 `set_queue` 后输出到 VM vnet 端口
- 内网流量：优先级更高的 `nw_src/nw_dst=<OVS 内网 CIDR>` 规则直接 `NORMAL` 放行

配置持久化仍使用 `virsh domiftune --config`，参数单位为 KB/s（average/peak）和 KB（burst）。OVS 下行队列按 `average` 平峰速率配置 `other-config:max-rate`，上传 meter 按 `average` 平峰速率配置 `rate`。

OVS 网络的运行态不会保留 `virsh domiftune --live` 的网卡 policing，启动或重刷限速时会先清掉 live 限速，再完全交给 OVS `meter/set_queue`，避免低速率上行出现明显抖动。

前端展示和用户操作使用 Mbps，后端自动转换：`1 Mbps = 125 KB/s`

核心服务文件：`server/service/bandwidth.go`

## 注意事项

1. **管理员不受限制**：管理员角色的 VM 不受配额约束，可自由设置任意速率
2. **配额为 0 表示不限制**：系统设置或用户配额设为 0 时，不应用任何限速
3. **关机 VM 的限速保存在 config 中**：即使 VM 关机，限速配置也会持久化到 VM 配置
4. **需要网卡和 IP 存在**：运行态 OVS 限速需要 VM 的 `vnet*` 端口和 OVS 内网 IP；推荐保持静态 DHCP 绑定
5. **只限制外网**：不要在 VM 端口上配置整口 QoS，否则会误伤内网流量
