# 防火墙

## 宿主机防火墙

「防火墙」页面的「宿主机防火墙」标签页用于直接控制宿主机 UFW。它与 KVM 网络防火墙分离：宿主机防火墙控制宿主机入站端口，KVM 网络防火墙继续控制虚拟机转发流量。

### 保护规则

开启宿主机防火墙前，后端会自动探测并要求确认以下端口：

- SSH 监听端口：优先通过 `sshd -T` 获取，失败时从 `ss -tlnp` 中识别。
- 面板运行端口：优先使用当前配置的 `KVM_PORT`，并结合当前监听端口校验。

启用后，SSH 和面板服务端口对应规则会被标记为保护规则，服务端会拒绝删除和编辑，避免误操作导致整机断连。VNC 默认范围 `5900-5999/tcp` 可通过页面一键添加，但它不是保护规则，后续允许编辑或删除。

### 规则管理

宿主机防火墙规则支持增删改查：

| 字段 | 说明 |
|------|------|
| 动作 | `allow` 或 `deny` |
| 协议 | `tcp`、`udp`，页面添加时也支持 TCP+UDP，会拆成两条规则 |
| 端口 | 支持单端口和范围端口，例如 `22` 或 `5900-5999` |
| 来源 CIDR | 留空表示任意来源；填写后仅匹配指定来源 |
| 备注 | 用于识别面板管理规则，面板自动规则以 `kvm-console:` 开头 |

规则状态以宿主机当前 UFW 规则为准，不新增数据库表。编辑规则时会先新增新规则，成功后再删除旧规则；如果新增失败，旧规则会保留。

### 端口转发联动

端口转发仍使用 `iptables` 写入 DNAT 和必要的 FORWARD 规则。无论宿主机防火墙当前是否启用，新增端口转发都会自动补一条 UFW 放通规则；这样下次开启宿主机防火墙时，已有端口转发不会被拦截。删除端口转发时，只清理面板自动创建的端口转发 UFW 规则，不会删除 SSH、面板、VNC 或用户手动创建的规则。

### Docker 兼容性

宿主机防火墙不会写入 Docker 链，不会修改 `DOCKER-USER`，启用时保持 UFW routed 默认允许。Docker bridge 模式不受宿主机防火墙约束，避免与当前 Docker 网络策略冲突。

### 连接管理

「连接管理」标签页支持：

- 关闭非防火墙端口连接：仅关闭本地端口不在当前 UFW 允许规则内的 TCP 已建立连接。
- 关闭全部连接：按字面尝试关闭所有 TCP 已建立连接，包括 SSH 和面板连接；执行后当前会话可能立即断开。

两个操作都会先展示预览并进行二次确认。

### 宿主机防火墙 API

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/firewall/host/status` | 获取 UFW 状态、规则、保护端口和推荐规则 |
| POST | `/api/firewall/host/enable/preview` | 预览启用宿主机防火墙时的自动放通端口 |
| POST | `/api/firewall/host/enable` | 启用宿主机防火墙，走任务队列，需要高风险验证 |
| POST | `/api/firewall/host/disable` | 关闭宿主机防火墙，走任务队列，需要高风险验证 |
| GET | `/api/firewall/host/rules` | 查询宿主机防火墙规则 |
| POST | `/api/firewall/host/rules` | 新增宿主机防火墙规则，需要高风险验证 |
| PUT | `/api/firewall/host/rules/:id` | 编辑宿主机防火墙规则，需要高风险验证 |
| DELETE | `/api/firewall/host/rules/:id` | 删除宿主机防火墙规则，需要高风险验证 |
| POST | `/api/firewall/host/rules/vnc-default` | 一键添加 VNC `5900-5999/tcp` 规则 |
| GET | `/api/firewall/host/connections/preview` | 预览将关闭的连接 |
| POST | `/api/firewall/host/connections/close` | 关闭连接，需要高风险验证 |

# KVM 全局网络防火墙

## 功能概述

网络防火墙用于统一管控 KVM 虚拟机的 IPv4 入站和出站转发流量。它是最上层的全局网络策略，作用于旧 OVS 内网和所有 VPC 交换机；不控制宿主机自身访问，也不控制 Docker 容器网络。

底层使用独立的 `nftables` 表：

```bash
table inet kvm_console_fw
```

系统不会修改 libvirt、Docker、UFW 自动生成的链。禁用或回滚时会删除该独立表，恢复到未管控状态。

## VPC ACL

VPC 安全组使用独立 nftables 表：

```bash
table inet kvm_console_vpc_acl
```

全局网络防火墙的优先级高于 VPC ACL，会先按区域、白名单、VM 覆盖和端口转发豁免做总控。VPC ACL 随后控制绑定到 VPC 交换机的 VM 入站访问，默认拒绝入站、允许出站，不同交换机之间默认隔离。管理员可在左侧「网络」里的「ACL 预览/应用」标签页查看生成规则并应用，高风险应用操作需要二次验证。

应用全局网络防火墙时，后端会自动把当前所有 VPC 交换机加入防火墙作用域：旧 OVS 使用 `br-ovs + VMSubnet`，VPC 交换机使用 `vpcswX + 交换机 CIDR`。因此出站区域限制、入站区域限制、VM 覆盖策略和端口转发豁免都会覆盖所有交换机。

端口转发流量 DNAT 到内部 VM 后仍会经过 VPC ACL。添加端口转发时，若 VM 已绑定 VPC 安全组，后端会自动添加对应入站允许规则。VPC 目标 IP 不再写入传统 `iptables FORWARD ACCEPT` 放行，历史遗留的 VPC 转发放行会在应用 VPC ACL 时自动清理，避免绕过安全组。

## 策略文件

| 路径 | 说明 |
|------|------|
| `/etc/kvm-console/firewall/policy.json` | 防火墙策略配置 |
| `/etc/kvm-console/firewall/rules.nft` | 后端生成的 nftables 规则 |
| `/etc/kvm-console/firewall/backups/` | 策略和规则备份，默认保留最近 10 份 |

默认部署后不会自动启用规则。管理员需要先保存策略、预览规则，再执行“应用规则”。

## 支持能力

### 全局策略

- **出站区域限制**：按目标 IP 所属区域限制 VM 访问外网。
- **入站区域限制**：按来源 IP 所属区域限制外部访问 VM 端口转发。
- **白名单 CIDR**：白名单优先级最高，命中后直接放行。
- **禁用 VM IPv6**：默认拒绝 `br-ovs` 上的 IPv6 转发，宿主机 IPv6 不受影响。
- **拦截动作**：默认 `reject`，也可切换为 `drop`。

### VM 覆盖

每台 VM 可设置：

| 模式 | 说明 |
|------|------|
| 继承全局 | 使用全局入站/出站策略 |
| 关闭管控 | 对该 VM 放行，不执行区域限制 |
| 仅允许入站 | 阻止该 VM 主动发起出站连接，仅保留外部连入后的回包 |
| 仅允许区域 | 该 VM 只允许指定区域访问/被访问 |
| 阻断区域 | 该 VM 阻断指定区域，其它区域放行 |

VM 覆盖依赖系统能解析到 VM IPv4 地址。建议重要 VM 先绑定静态 IP。
“仅允许入站”会阻断 VM 主动访问外部网络，也会影响 VM 内主动 DNS、软件源、时间同步等请求；如果业务依赖这些能力，请改用区域限制或白名单策略。

### 端口转发豁免

端口转发列表会显示“入站区域限制”开关：

- 开启：该转发继承全局入站区域限制。
- 关闭：该转发对应的 VM 目标 IP、协议和目标端口被豁免入站区域限制。

已有端口转发首次启用防火墙时默认继承全局策略，可手动关闭单条豁免。

## 区域数据

区域数据支持两种来源：

1. **本地导入**：粘贴 IPv4 CIDR 列表，适合自维护数据或内网自定义区域。
2. **在线更新**：默认使用 IPdeny 聚合 CIDR zone 文件，下载地址格式：

```text
https://www.ipdeny.com/ipblocks/data/aggregated/{code}-aggregated.zone
```

例如 `cn` 会下载 `cn-aggregated.zone`。

> 使用在线数据时请遵守 IPdeny 的版权和使用限制。若生产环境无法访问外网，可只使用本地导入。

## API

页面切换到「KVM 网络防火墙」标签页时会读取 `/api/firewall/status` 回填已保存策略、nftables 生效状态和 VM 列表；刷新页面后即使默认停留在「宿主机防火墙」标签页，再切回 KVM 标签页也会重新拉取后端状态，避免显示前端默认策略。

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/firewall/status` | 获取状态、策略和 VM 列表 |
| GET | `/api/firewall/policy` | 获取策略 |
| PUT | `/api/firewall/policy` | 保存策略，不应用 |
| POST | `/api/firewall/preview` | 预览 nftables 规则 |
| POST | `/api/firewall/apply` | 应用规则，走任务队列，需要高风险验证 |
| POST | `/api/firewall/disable` | 禁用规则，走任务队列，需要高风险验证 |
| POST | `/api/firewall/rollback` | 回滚规则，走任务队列，需要高风险验证 |
| POST | `/api/firewall/geoip/import` | 本地导入区域 CIDR |
| POST | `/api/firewall/geoip/update` | 在线更新区域 CIDR，走任务队列 |
| PUT | `/api/firewall/port-forward` | 设置端口转发入站区域限制豁免 |

## 手工回滚

如应用后网络异常，可在宿主机执行：

```bash
nft delete table inet kvm_console_fw
```

或使用项目脚本：

```bash
bash scripts/firewall-control.sh disable
```

生产环境启用前建议保存以下信息：

```bash
nft list ruleset > /root/nft.ruleset.before-kvm-firewall.txt
iptables-save > /root/iptables.before-kvm-firewall.txt
ufw status verbose > /root/ufw.before-kvm-firewall.txt
```

## 注意事项

1. 当前版本只生成 IPv4 区域规则，IPv6 转发默认被拒绝。
2. Docker 容器网络不纳入本功能范围。
3. 管理员保存策略后不会立即影响网络，只有点击“应用规则”后才会写入 nftables。
4. 在线 GeoIP 更新失败时会保留旧数据，不会应用半成品规则。
