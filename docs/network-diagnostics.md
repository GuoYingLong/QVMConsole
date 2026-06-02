# 网络抓包与诊断工具使用文档

## 功能说明

管理员可在虚拟机「网络管理」弹窗的「网络诊断」标签页中，对当前 VM 的运行态 `vnet*` / `tap*` OVS 端口进行临时抓包。

该功能只读取运行态网络信息，不修改 VM XML、不修改 OVS 流表、不重启网络服务。抓包任务通过任务队列执行，可在任务中心查看进度并取消。

## 抓包能力

- 选择当前 VM 的运行态接口。
- 按协议、源 IP、目标 IP、任意端口、源端口、目标端口生成 BPF 过滤条件。
- 提供 ARP、DHCP、DNS、当前 VM IP、端口转发入站流量快捷模板。
- 页面轮询显示 `tcpdump` 文本摘要。
- 抓包完成后可下载 pcap 文件。
- 页面可删除当前 pcap 文件；删除后任务摘要仍保留。

## 安全限制

- 仅管理员可用。
- 开始抓包需要高风险二次验证。
- 不支持自由输入原始 `tcpdump` 命令，只允许结构化过滤条件。
- 后端会限制抓包时长、包数和文件大小，避免长时间占用宿主机资源。
- 抓包文件只保存在后端配置的抓包目录中，下载接口会校验任务 ID 和文件路径。
- 每次提交新抓包时，系统会自动删除上一份已生成的 pcap 文件，避免抓包文件持续堆积。

## 配置项

| 配置项 | 默认值 | 说明 |
|---|---:|---|
| `KVM_NETWORK_CAPTURE_DIR` | `/var/lib/kvm-console/captures` | pcap 文件保存目录 |
| `KVM_NETWORK_CAPTURE_DEFAULT_SECONDS` | `30` | 未指定时的默认抓包时长 |
| `KVM_NETWORK_CAPTURE_MAX_SECONDS` | `120` | 最大抓包时长 |
| `KVM_NETWORK_CAPTURE_MAX_MB` | `64` | 单次抓包文件大小上限 |
| `KVM_NETWORK_CAPTURE_MAX_PACKETS` | `5000` | 单次抓包包数上限 |

## API

管理员接口：

- `GET /api/vm/:name/network/diagnostics`：读取 VM 网络诊断摘要、可抓包接口、邻居表和快捷模板。
- `POST /api/vm/:name/network/capture`：提交抓包任务，需要二次验证。
- `GET /api/network/captures/:task_id`：读取抓包 session、摘要、文件状态。
- `GET /api/network/captures/:task_id/download`：下载 pcap 文件。
- `DELETE /api/network/captures/:task_id`：删除 pcap 文件，保留任务摘要。

抓包请求示例：

```json
{
  "interface_name": "vnet3",
  "filter": {
    "protocol": "tcp",
    "source_ip": "",
    "dest_ip": "192.168.122.10",
    "port": 0,
    "source_port": 0,
    "dest_port": 22
  },
  "duration_seconds": 30,
  "max_mb": 64,
  "max_packets": 5000
}
```

## 常用诊断

- ARP 异常：选择「ARP」模板，观察是否有 VM IP 对应的 ARP 请求和应答。
- DHCP 异常：选择「DHCP」模板，观察 UDP 67/68 是否有请求和响应。
- DNS 异常：选择「DNS」模板，观察 53 端口是否有查询和响应。
- 端口转发异常：选择对应端口转发模板，观察流量是否到达 VM 端口。

## 依赖

本功能新增宿主机 apt 依赖：

- `tcpdump`

安装脚本会自动检查并安装该依赖。手动安装：

```bash
apt-get update
apt-get install -y tcpdump
```
