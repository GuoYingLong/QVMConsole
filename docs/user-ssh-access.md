# SSH 访问控制

## 功能说明

面板创建的普通用户在系统中会自动创建对应的系统账户。为了安全起见，这些用户的 **SSH 登录默认关闭**。

管理员可以在面板的 **用户管理** 页面中，通过每个用户的 SSH 开关来控制其是否允许通过 SSH 登录宿主机。

## 工作原理

面板通过 **双重机制** 控制用户的 SSH 访问：

### 1. Shell 切换（主要机制）
- **SSH 关闭** → 系统用户的 shell 设为 `/usr/sbin/nologin`，直接禁止登录
- **SSH 开启** → 系统用户的 shell 设为 `/bin/bash`，允许正常登录

### 2. sshd DenyUsers（辅助机制）
- 通过 `/etc/ssh/sshd_config.d/kvm-console-deny.conf` 文件管理 SSH 拒绝列表
- 使用 `DenyUsers` 指令列入所有被禁止的用户

### 切换流程
每次切换开关时，面板会自动：
1. 更新数据库中的 `ssh_enabled` 状态
2. 切换系统用户的 shell（`nologin` ↔ `bash`）
3. **如果是关闭 SSH**，立即杀死该用户的所有现有 SSH 会话（已连接的会被断开）
4. 重新生成 `/etc/ssh/sshd_config.d/kvm-console-deny.conf`
5. 热加载 sshd 配置（`systemctl reload sshd`）

## 管理操作

### 在面板中操作

1. 登录管理员账户
2. 进入 **用户管理** 页面
3. 在用户列表的 **SSH** 列中，找到目标用户
4. 切换开关即可开启或关闭该用户的 SSH 访问

### 注意事项

- **管理员用户**不受此限制，SSH 列显示为 `-`
- 新创建的用户 SSH 默认**关闭**
- 服务启动时会自动同步 SSH 配置，确保数据库状态与系统 sshd 配置一致
- 删除用户时，SSH 拒绝列表会自动更新

## 系统依赖

- 需要 sshd 支持 `Include` 指令（通常满足，Ubuntu 20.04+ / Debian 10+ 默认支持）
- 如果 `/etc/ssh/sshd_config` 中没有 `Include /etc/ssh/sshd_config.d/` 指令，面板会自动添加
