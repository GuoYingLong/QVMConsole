# API Key 与外部接口调用说明

## 功能概述

右上角账户菜单进入“安全设置”后，新增 `API` 标签页。用户可以生成一组 API ID 和 API Key，用于外部程序调用后端接口。

API Key 属于当前登录用户：

- 管理员 Key 拥有管理员账号本身可访问的接口权限
- 普通用户 Key 仍受 VM 归属、轻量云限制、配额和功能开关限制
- 账号被禁用或未激活时，API Key 不可继续使用
- 重新生成后，旧 API Key 会立即失效
- API Key 明文只在生成时显示一次，后端只保存哈希

## 后端接口

### 查看 API Key 状态

`GET /api/auth/api-key`

需要登录 Token 或 API Key。

响应示例：

```json
{
  "code": 200,
  "message": "ok",
  "data": {
    "api_key_id": "kvm_id_xxx",
    "key_prefix": "kvm_sk_xx...abcd",
    "created_at": "2026-05-01T13:00:00+08:00",
    "last_used_at": null,
    "enabled": true
  }
}
```

### 生成或轮换 API Key

`POST /api/auth/api-key`

需要登录 Token。该操作属于敏感操作，会触发高风险二次验证。

响应示例：

```json
{
  "code": 200,
  "message": "API 凭证已生成，请立即复制保存 API Key",
  "data": {
    "api_key_id": "kvm_id_xxx",
    "api_key": "kvm_sk_xxx",
    "key_prefix": "kvm_sk_xx...abcd",
    "created_at": "2026-05-01T13:00:00+08:00",
    "last_used_at": null,
    "enabled": true
  }
}
```

### 撤销 API Key

`DELETE /api/auth/api-key`

需要登录 Token。该操作属于敏感操作，会触发高风险二次验证。

## 外部调用认证

推荐使用请求头：

```bash
curl -H "X-API-Key-ID: kvm_id_xxx" \
  -H "X-API-Key: kvm_sk_xxx" \
  "http://127.0.0.1:8080/api/vm/list"
```

也兼容单个 Authorization 请求头：

```bash
curl -H "Authorization: ApiKey kvm_id_xxx:kvm_sk_xxx" \
  "http://127.0.0.1:8080/api/vm/list"
```

登录、邀请注册、找回密码、安全初始化、邮箱绑定和 2FA 绑定/关闭等账户安全流程只接受 JWT 流程令牌，不接受 API Key。

## 高风险操作

API Key 不会绕过二次验证。删除虚拟机、重置密码、修改防火墙、修改宿主机级配置等接口仍会返回 `428`。

调用方需要：

1. 收到 `428` 后读取响应里的 `operation`、`method`、`challenge_id`
2. 调用 `POST /api/auth/high-risk/verify` 完成 TOTP 或邮箱验证
3. 对原请求追加 `X-High-Risk-Token`

示例：

```bash
curl -X POST "http://127.0.0.1:8080/api/auth/high-risk/verify" \
  -H "X-API-Key-ID: kvm_id_xxx" \
  -H "X-API-Key: kvm_sk_xxx" \
  -H "Content-Type: application/json" \
  -d '{"method":"totp","code":"123456","operation":"delete_vm"}'
```

## 后续开发规则

后续新增后端业务接口时，除登录、注册、邀请、找回密码、安全初始化、邮箱、2FA 等账户安全流程外，必须默认兼容 API Key 调用。

新增接口时需要同步检查：

- 路由是否挂在兼容 API Key 的认证中间件下
- 普通用户权限、管理员权限、VM 归属、轻量云限制是否仍然生效
- 敏感操作是否继续使用高风险二次验证
- 前端接口文档页面是否补充新接口
- `docs/backend-api.md` 或对应功能文档是否补充调用说明
