# 虚拟机 SMBIOS 类型 1 设置

## 功能说明

虚拟机高级设置中新增了 `SMBIOS` 配置入口，用于设置 SMBIOS Type 1（System Information）字段，向来宾系统暴露一组虚拟机器身份信息。

当前支持字段：

- `base64`
- `family`
- `manufacturer`
- `product`
- `serial`
- `sku`
- `uuid`
- `version`

对应的 libvirt XML 形式为：

- `<sysinfo type='smbios'>`
- `<system><entry name='...'>...</entry></system>`
- `<os><smbios mode='sysinfo'/></os>`

## 适用范围

- 普通 ISO 创建
- 从模板克隆
- 从磁盘导入
- 编辑已有虚拟机

## 使用建议

- 普通业务虚拟机通常保持默认，不建议随意修改
- 需要模拟特定硬件身份、兼容旧授权机制、对接资产识别系统时再使用
- 若只是需要让来宾看到固定 UUID，建议优先在新建、导入、克隆时一次性设置

## 字段说明

### base64

- 作用：控制面板是否将填写值按 Base64 解码后再写入
- 使用场景：兼容外部配置或需要传递特殊字符时
- 注意：这只是写入辅助开关，不会以 Base64 形式保存在 libvirt XML 中

### manufacturer / product / version / serial / sku / family

- 作用：设置 SMBIOS Type 1 的基础身份字段
- 典型用途：
  - 模拟厂商与产品名称
  - 固定序列号
  - 兼容依赖 SMBIOS 识别逻辑的软件

### uuid

- 作用：设置 SMBIOS Type 1 UUID
- 格式：标准 UUID
- 重要限制：
  - libvirt 要求 SMBIOS UUID 与虚拟机顶层 UUID 保持一致
  - 已存在虚拟机若直接改 UUID，libvirt 会拒绝 `define`
  - 因此当前面板只在新建、导入、克隆时允许显式填写 UUID
  - 编辑已有虚拟机时，页面仅展示当前 UUID，不支持直接修改

## 生效规则

- 新建 / 导入 / 克隆：在生成或组装 libvirt XML 时直接写入
- 编辑已有虚拟机：更新持久 XML 配置
- 若虚拟机已经在运行中，通常需要重启后才会看到来宾中的 SMBIOS 变化

## 测试结论

在测试环境 `test` 虚拟机上已验证：

- 使用 `<sysinfo type='smbios'>` + `<os><smbios mode='sysinfo'/></os>` 可以正常定义
- SMBIOS UUID 与顶层 `<uuid>` 不一致时，libvirt 会拒绝定义
- 因此实现中对 UUID 做了保护，避免编辑已有虚拟机时直接触发定义失败

## 相关位置

- 前端：`web/src/components/VmForm.vue`
- 后端解析与写入：`server/service/vm_smbios.go`
- 高级设置总说明：`docs/vm-advanced-settings.md`
