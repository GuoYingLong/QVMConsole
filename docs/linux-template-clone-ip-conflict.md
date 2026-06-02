# Linux 模板并发克隆 IP 冲突修复

## 背景

从同一个 Linux 模板同时创建两台或更多虚拟机时，如果模板磁盘中保留了相同的 `machine-id`、DHCP 租约或 cloud-init 实例缓存，来宾系统首次启动时可能使用相同的 DHCP client-id/DUID 请求地址，导致两台虚拟机首次获取到同一个 IP。

旧流程是在虚拟机启动并获取 IP 后，再通过 SSH 进入系统重置 `machine-id` 和 DHCP 租约。并发克隆时多个任务可能先拿到同一个 IP，后续 SSH 初始化会连接到同一台来宾，从而出现账号、密码、hostname 或网络状态互相覆盖。

## 当前处理

Linux 模板克隆现在会在克隆盘创建后、虚拟机首次启动前，通过 `virt-customize --no-network` 对克隆盘离线执行以下处理：

- 清空 `/etc/machine-id`
- 删除 `/var/lib/dbus/machine-id`
- 清理常见 DHCP 租约缓存
- 清理 cloud-init 实例缓存
- 写入本次创建请求中的 hostname
- 更新 `/etc/hosts` 中的 `127.0.1.1` hostname 映射

这样每台克隆机在第一次启动时都会生成自己的系统身份，再发起 DHCP 请求，避免并发创建时首次 IP 冲突。后续 SSH 初始化仍负责设置用户、密码和磁盘扩容。

## 兼容性

- 仅影响 `linux` 类型模板克隆。
- Windows、FnOS、Other 类型逻辑不变。
- 导入虚拟机等未经过模板克隆离线预处理的路径，仍保留原有 SSH 初始化中的在线 `machine-id` 重置和网络刷新逻辑。
- 本次修复复用既有 `virt-customize` 依赖，没有新增 apt 或第三方依赖，因此无需修改 `docs/dependencies.md`。

## 已经出现重复 IP 的处理建议

如果测试机或生产环境中已经存在重复 IP 的 Linux 克隆机，建议通过 VNC 或单台关机后操作的方式逐台修复，避免继续通过重复 IP SSH 到错误虚拟机：

1. 先保留其中一台运行，其他重复 IP 虚拟机关机。
2. 在运行中的虚拟机内清空 `machine-id` 和 DHCP 租约，重启网络或重启虚拟机。
3. 确认该虚拟机 IP 唯一后，再处理下一台。

如果允许关机，优先关机后离线处理磁盘，再启动确认新 IP，这是代价更低且风险更小的方式。

## 桥接模式 IP 获取

### 问题

桥接（直通）模式下，虚拟机的 DHCP 请求直接发送到上游物理路由器，由路由器分配 IP。宿主机仅作为二层转发，**没有 DHCP 租约记录**，因此 `virsh domifaddr --source lease`、OVS dnsmasq 租约、VPC 静态绑定等方式无法获取 IP。

`virsh domifaddr --source agent` 依赖客户机安装 `qemu-guest-agent`，如果模板未预装也同样无法工作。

结果是 `getVMIP()` 兜底也无法获取 IP，克隆流程卡在"等待虚拟机启动"，最终超时报错。

### 解决方案：主动 ARP 扫描

`getVMIP()` 方式8 新增了主动 ARP 扫描：

1. 通过 `virsh domiflist` 获取虚拟机所在的桥接接口名称
2. 通过 `ip addr` 获取该桥接接口的 IPv4 子网 CIDR
3. 优先使用 `arp-scan --interface=<bridge> --localnet` 直接获取 MAC-IP 映射表
4. 如果 `arp-scan` 不可用，回退使用 `nmap -sn <cidr>` 发送 ARP 探测填充宿主机 ARP 表，再通过 `ip neigh` 匹配 MAC 获取 IP
5. 同一虚拟机每 12 秒最多扫描一次，避免频繁扫描

### 依赖

- `arp-scan`：首选，速度快精度高，建议安装
- `nmap`：备选，兼容性好，`arp-scan` 不可用时自动回退

两者都已在 `install.sh` 和 `dependencies.md` 中登记。

### 兼容性

- 不影响 NAT（OVS）模式下的现有流程（方式1-7 已能获取 IP 时不会触发扫描）
- 不影响 Windows / FnOS / Other 类型
- 扫描有节流限制，不会在每次 `getVMIP()` 调用时触发
