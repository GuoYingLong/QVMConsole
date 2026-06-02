# 安装脚本使用说明

本文档说明 `install.sh` 的安装、更新和卸载行为，适用于全新 Debian / Ubuntu 系列宿主机。

## 运行模式

首次运行脚本时会进入安装流程。脚本检测到 `/opt/kvm-console/kvm-console` 或 `kvm-console.service` 已存在时，会进入菜单：

```text
1. 更新
2. 卸载
```

更新会保留已有数据库、`.env` 和业务配置，只替换后端二进制与 `web-dist`，并重新检测依赖、目录、systemd、OVS、Project Quota 等运行地基。卸载会停止并禁用面板服务，默认保留数据库、配置、虚拟机磁盘、模板、libvirt 定义和用户存储镜像。

## 硬件虚拟化检测

脚本会在安装和更新流程中检测 CPU 是否存在 KVM 硬件虚拟化标记：

- Intel：`vmx`
- AMD：`svm`

未检测到硬件虚拟化标记时脚本会拒绝继续安装。安装依赖后还会尝试加载 `kvm`、`kvm_intel` 或 `kvm_amd` 模块，并确认 `/dev/kvm` 可用；如果不可用，需要先在 BIOS/UEFI 中开启虚拟化，或确认上层宿主机已开放嵌套虚拟化。

## 安装和更新会检查的地基

脚本每次安装或更新都会执行以下检测与补齐：

| 类型 | 内容 |
|------|------|
| apt 依赖 | QEMU、libvirt、OVS、dnsmasq、guestfs、quota、nftables、iptables、UFW、SSH、curl 等 |
| 系统命令 | `virsh`、`qemu-img`、`virt-install`、`virt-customize`、`guestfish`、`ovs-vsctl`、`dnsmasq`、`nft`、`iptables`、`setquota`、`repquota` 等 |
| 核心服务 | `libvirtd`、`openvswitch-switch`、`ssh/sshd` |
| 运行目录 | `/opt/kvm-console`、模板/导入/导出/ISO（默认 `/var/lib/libvirt/images/ISO`）/克隆目录、`/etc/kvm-console`、`/etc/kvm-portforward`、`/etc/libvirt/vm-access` 等 |
| 用户存储 | `/var/lib/kvm-user-storage.img`、`/var/lib/kvm-user-storage`、`/etc/fstab`、`/etc/projects`、`/etc/projid` |
| OVS 网络 | `br-ovs`、网关地址、dnsmasq 配置、`kvm-console-ovs-dnsmasq.service`、IPv4 转发、基础 NAT/FORWARD 规则和本机 DHCP/DNS 入站规则 |
| 面板服务 | `/etc/systemd/system/kvm-console.service`、`network-online.target`、`libvirtd` 和 OVS 依赖 |

## 配置合并策略

首次安装会生成完整 `.env`。更新时脚本只会追加缺失的 `KVM_*` 配置项，不覆盖已有生产值；端口会按用户输入更新。

旧版本升级时如果缺少 `KVM_VM_CREDENTIAL_SECRET` 或 `KVM_SECURITY_SECRET`，脚本会补为空值，让后端继续回退使用 `KVM_JWT_SECRET`，避免历史 VM 凭据或安全配置无法解密。全新安装会生成独立随机密钥。

## OVS 出口网卡

`KVM_OVS_UPLINK` 留空时，脚本通过默认路由自动检测出口网卡。若机器没有默认路由，脚本会继续安装并提示在 `/opt/kvm-console/.env` 中手动配置 `KVM_OVS_UPLINK`，之后可在面板 OVS 诊断中执行修复。

安装和更新时会为 OVS 基础网桥与已存在的 `vpcsw*` 网关口补齐 dnsmasq 入站规则：UDP 67、UDP 53、TCP 53。这样在宿主机 INPUT 默认 DROP 或启用 UFW 的情况下，虚拟机仍能向宿主机上的 dnsmasq 获取 DHCP 地址和 DNS 响应。

## 安全建议

首次安装完成后请尽快完成以下操作：

1. 登录默认管理员 `admin / admin123` 后修改密码。
2. 配置 SMTP 与 `KVM_PUBLIC_BASE_URL`，确保邮箱验证、找回密码和高风险二次验证可用。
3. 为管理员绑定邮箱并启用 2FA。
4. 在生产环境中确认 UFW、nftables、iptables 与已有防火墙策略不会冲突。
