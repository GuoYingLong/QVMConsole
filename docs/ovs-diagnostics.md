# OVS 网络诊断使用文档

## 功能说明

`OVS 网络` 页面用于管理员查看当前宿主机的 Open vSwitch 网络状态，重点覆盖：

- OVS 网桥、网关 IP、内网 CIDR、出口网卡。
- `openvswitch-switch` 与 `kvm-console-ovs-dnsmasq.service` 服务状态。
- `net.ipv4.ip_forward`、NAT 和 FORWARD 规则状态。
- OVS 端口、ofport、关联 VM、MAC、IP 和异常端口。
- OVS DHCP 静态绑定、租约和冲突检测。
- VM 网络管理弹窗中的单台 VM 运行状态、IP 来源和 OVS 限速只读状态。

本功能不把 VM 网络运行态写入数据库，所有状态优先通过命令和 dnsmasq 文件读取，便于与已经运行的虚拟机保持同步。

## API

管理员接口：

- `GET /api/ovs/status`：读取 OVS 基础状态。
- `GET /api/ovs/ports`：读取 OVS 端口和异常信息。
- `GET /api/ovs/leases`：读取静态绑定、DHCP 租约和冲突。
- `POST /api/ovs/check`：执行只读健康检查。
- `POST /api/ovs/repair`：提交 OVS 修复任务，需要二次验证。

VM 访问权限接口：

- `GET /api/vm/:name/network/status`：读取单台 VM 的 OVS 网络运行状态。

## 修复流程

一键修复只调用现有 `EnsureOVSNetworkReady` 能力，修复内容包括：

- 确保 OVS 网桥存在并处于 UP 状态。
- 确保网桥配置预期网关地址。
- 写入并启动 OVS dnsmasq 配置和 systemd 服务。
- 开启 IPv4 转发。
- 补齐 OVS NAT、FORWARD 与 dnsmasq 本机 DHCP/DNS 入站规则。

修复不会删除未知 iptables 规则，不会重建 VM 网卡，也不会主动对运行中的 VM 做拔插网卡操作。

## 常用排障命令

```bash
ovs-vsctl show
ovs-ofctl -O OpenFlow13 show br-ovs
ovs-ofctl -O OpenFlow13 dump-flows br-ovs
ip -4 addr show dev br-ovs
systemctl is-active openvswitch-switch
systemctl is-active kvm-console-ovs-dnsmasq.service
sysctl -n net.ipv4.ip_forward
cat /etc/kvm-console/ovs/dhcp-hosts
cat /var/lib/kvm-console/ovs/dnsmasq.leases
iptables -t nat -S POSTROUTING
iptables -S FORWARD
virsh domiflist <vm-name>
virsh dumpxml <vm-name>
```

## 依赖说明

本阶段不新增依赖，沿用已有依赖：

- `openvswitch-switch`
- `dnsmasq-base`
- `iptables`
- `systemd`
- `libvirt` / `virsh`

依赖安装说明仍以 `docs/dependencies.md` 为准。
