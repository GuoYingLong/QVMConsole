# 虚拟机凭据展示与 Linux/Windows/fnOS 重置密码

## 功能范围

- 创建虚拟机时，如果流程中填写了登录用户名和密码，系统会在任务成功后自动保存凭据。
- 当前已接入的创建流程：
  - 模板克隆
  - 磁盘导入
- 虚拟机详情页会展示已保存的用户名和密码，并支持复制。
- 新增 `Linux / Windows / fnOS` 虚拟机重置密码能力。

## 详情页说明

- 详情页新增“登录凭据”卡片。
- 会展示：
  - 用户名
  - 密码
  - 来源（模板克隆 / 磁盘导入 / 密码重置）
  - 最近更新时间
  - 最近一次重置时间
- 页面中的复制按钮已兼容 `HTTP` 场景，会自动降级到传统复制方式。

## 重置密码规则

- 入口：虚拟机详情页 → 登录凭据 → `重置登录密码`
- 当前支持 `Linux`、`Windows` 和 `fnOS` 虚拟机。
- 必须先将虚拟机关机，否则后端会直接拒绝执行。
- 不需要知道旧密码，后端会通过离线方式修改或注入磁盘中的系统密码重置流程。
- 对于 `fnOS`，这里重置的是离线注入的本地管理员账号密码，克隆后用于网页登录的管理员密码也会同步更新。
- 对于 `Windows`：
  - 表单中的默认账号为 `administrator`
  - `Windows Server` 修改用户名通常无效，建议保持默认
  - 任务成功后还需要手动开机一次，系统会自动完成密码重置并自动关机
- 新密码沿用现有强密码规则：
  - 长度至少 `12` 位
  - 必须同时包含大写字母、小写字母、数字和符号
  - 允许符号：`!@#$%^&*_-+=?`

## 兼容性边界

Linux/fnOS 的离线重置能力适用于常见的本地账号密码登录型系统，但不能承诺覆盖所有发行版。

通常可兼容：

- `Ubuntu`
- `Debian`
- `CentOS`
- `Rocky Linux`
- `AlmaLinux`
- `openEuler`
- 其他使用本地 `/etc/passwd`、`/etc/shadow` 的常规 Linux

以下场景不保证支持：

- 磁盘加密（如 `LUKS`）
- 外部认证（如 `LDAP` / `AD` / `SSSD`）
- 不可变系统或只读系统
- SSH 禁用密码登录

> 注意：即使系统密码已成功重置，如果虚拟机内部 SSH 明确禁止密码登录，仍然无法通过 SSH 密码方式登录。

Windows 兼容方案依赖宿主机现有 `libguestfs-tools`，通过离线写入一次性 `SYSTEM\Setup` 脚本完成。若 Windows 磁盘因异常断电、休眠或未完全卸载导致文件系统只读，任务可能失败，此时建议先正常关机后再重试。

## 后端实现说明

- 凭据保存在数据库表 `vm_credentials`
- 数据库存储的是加密后的密码，不是明文
- 默认使用 `KVM_VM_CREDENTIAL_SECRET` 作为加密密钥
- 如果未设置 `KVM_VM_CREDENTIAL_SECRET`，则自动回退为 `KVM_JWT_SECRET`

## 接口说明

### 详情接口

- `GET /api/vm/:name`
- `GET /api/vm/:name/sse`

返回数据中新增：

```json
{
  "credential": {
    "username": "xiaozhu",
    "password": "ResetAa12345!",
    "source": "password_reset",
    "operator": "admin",
    "updated_at": "2026-04-25 18:20:00",
    "last_reset_at": "2026-04-25 18:20:00"
  }
}
```

### Linux/Windows/fnOS 重置密码接口

- `POST /api/vm/:name/password/reset`

请求体：

```json
{
  "username": "xiaozhu",
  "password": "ResetAa12345!"
}
```

返回：

- 成功后返回任务 ID，实际执行过程请在任务中心查看

## 测试结论

已在测试机通过 MCP 验证以下流程：

1. Linux/fnOS：
   - 找到测试虚拟机磁盘
   - 关机虚拟机
   - 在未知旧密码的情况下离线重置密码
   - 重新开机并确认新密码可用
2. Windows：
   - 在测试机 `wintest` 上确认可进入 `Administrator` 登录界面
   - 验证 `SYSTEM\Setup` 一次性脚本触发方式可以拉起重置脚本窗口
   - 最终实现采用“关机后离线注入，手动开机一次自动处理并自动关机”的产品策略

因此当前实现统一采用“先关机，再离线处理”的产品策略，不支持运行中直接修改系统密码。
