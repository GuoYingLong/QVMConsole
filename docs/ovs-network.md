# OVS 网络使用文档

## 拓扑说明

新平台默认使用独立 Open vSwitch 网桥 `br-ovs`：

- 宿主机物理网卡不迁移，原 IP、默认路由和 netplan 配置保持不变
- VM 网卡使用 libvirt `bridge` 接口接入 `br-ovs`，并写入 `virtualport type='openvswitch'`
- `br-ovs` 使用 `192.168.122.1/24` 作为内网网关
- VM 通过宿主机 iptables NAT 从默认路由网卡出站
- DHCP 与静态 IP 由独立 dnsmasq 管理，不再使用 libvirt `default` 网络

## 配置项

| 配置项 | 默认值 | 说明 |
|---|---:|---|
| `KVM_NETWORK_BACKEND` | `ovs` | 当前仅支持 OVS |
| `KVM_OVS_BRIDGE` | `br-ovs` | VM 接入的 OVS 网桥 |
| `KVM_OVS_UPLINK` | 空 | NAT 出口网卡，留空自动读取默认路由 |
| `KVM_SUBNET_PREFIX` | `192.168.122` | OVS 内网网段前缀 |
| `KVM_OVS_DHCP_START` | 空 | 留空时使用 `${KVM_SUBNET_PREFIX}.2` |
| `KVM_OVS_DHCP_END` | 空 | 留空时使用 `${KVM_SUBNET_PREFIX}.254` |

## 自动创建内容

后端在启动 VM、创建 VM、导入 VM、克隆 VM、绑定静态 IP 或添加端口转发前会自动确保：

- `openvswitch-switch` 服务可用
- `br-ovs` 已存在并配置网关 IP
- `/etc/kvm-console/ovs/dnsmasq.conf` 与 `kvm-console-ovs-dnsmasq.service` 已写入
- `/etc/kvm-console/ovs/prepare-bridge.sh` 已写入，dnsmasq 服务启动前会先补建并启动 `br-ovs`，避免开机时因网桥尚未创建而启动失败
- `/etc/kvm-console/ovs/dhcp-hosts` 用于静态 IP
- `/var/lib/kvm-console/ovs/dnsmasq.leases` 用于 DHCP 租约
- `net.ipv4.ip_forward=1`
- `POSTROUTING MASQUERADE` 与 OVS 转发规则已补齐

## 后端重启影响

面板后端启动时会做 OVS/VPC 运行态补齐，但不会无条件重启运行中的 VM 或反复重启网络服务：

- `openvswitch-switch` 已运行时只检查状态，不重复 start。
- OVS DHCP systemd unit、dnsmasq 配置和预启动脚本内容未变化时，不重启 `kvm-console-ovs-dnsmasq.service`。
- VPC 交换机 dnsmasq 进程仍在运行且配置未变化时，不 kill/restart，只在缺失或配置变化时补启动。
- VM 已经写入目标 VLAN 时，不重复 `virsh define`；运行态 `vnet*` 端口已有目标 tag 时，不重复 set。
- 后端重启只补齐缺失的网桥、网关、NAT、ACL、端口转发和带宽规则，不会主动断开 VM 网卡。

因此正常更新面板后端时，VM 现有网络连接不应因为服务进程重启而被中断；只有 OVS/DHCP/VPC 配置确实发生变化或运行态缺失时，才会进行对应的轻量修复。

## 静态 IP 约束

静态 DHCP 绑定必须保持 VM、MAC、IP 一一对应：

- 同一个 IP 不能绑定给多个 MAC 或多个 VM
- 同一个 MAC 不能绑定多个 IP 或多个 VM
- 如果 VM 的 MAC 被修改，系统会按 VM 名称把原静态 IP 迁移到当前 MAC，而不是重新分配新 IP
- 如果目标 IP 已经属于其他 VM，绑定会被拒绝，避免通过修改 MAC 绕过限制或抢占地址

## 外网限速

VM 运行态限速使用 OVS `linux-htb` QoS 队列、OpenFlow `set_queue` 和 OpenFlow meter，不对 `br-ovs` 或 VM 端口做整口限速：

- VM 上行外网：匹配 VM 的 `vnet*` OVS 端口和 VM 内网 IP，目的地址不属于 OVS 内网 CIDR 时进入上传 meter，再输出到宿主机 `LOCAL`
- VM 下行外网：匹配 OVS `LOCAL` 入方向和 VM 内网 IP，来源地址不属于 OVS 内网 CIDR 时进入下行队列
- VM 与宿主机、VM 与 VM 的内网流量优先 `NORMAL` 放行，不参与外网限速
- 每次设置限速前会按 VM 名称清理旧 flow/QoS/Queue/meter，避免改 IP 或改 MAC 后残留旧规则

## 验证命令

```bash
ovs-vsctl show
ip -br addr show br-ovs
systemctl status kvm-console-ovs-dnsmasq.service
cat /var/lib/kvm-console/ovs/dnsmasq.leases
iptables -t nat -S POSTROUTING | grep 192.168.122.0/24
iptables -S FORWARD | grep br-ovs
virsh domiflist <vm-name>
ovs-ofctl -O OpenFlow13 dump-flows br-ovs | grep set_queue
ovs-ofctl -O OpenFlow13 dump-flows br-ovs | grep meter
ovs-ofctl -O OpenFlow13 dump-meters br-ovs
ovs-vsctl list qos
ovs-vsctl list queue
```

VM 的网卡应显示为 `Type bridge`、`Source br-ovs`，`ovs-vsctl show` 中应能看到对应 `vnet*` 端口。

## 回滚测试配置

如需在测试机上撤销 OVS 验证配置：

```bash
systemctl disable --now kvm-console-ovs-dnsmasq.service
ovs-vsctl --if-exists del-br br-ovs
iptables -t nat -D POSTROUTING -s 192.168.122.0/24 -o <出口网卡> -j MASQUERADE
iptables -D FORWARD -i br-ovs -o <出口网卡> -j ACCEPT
iptables -D FORWARD -i <出口网卡> -o br-ovs -m conntrack --ctstate RELATED,ESTABLISHED -j ACCEPT
```

如果曾临时修改测试 VM XML，请用备份 XML 执行：

```bash
virsh define /root/<vm-name>-pre-ovs-*.xml
```
