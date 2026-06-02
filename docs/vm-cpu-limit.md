# 虚拟机 CPU 限制

## 功能说明

虚拟机表单的“CPU 与内存”区域新增 `CPU 限制`，仅管理员可见。该字段使用百分比表达虚拟机可使用的 CPU 总能力，默认是“无限制”。

## 交互规则

- 默认关闭，表示不写入 CPU 限速。
- 开启后可输入 `1-100` 的百分比。
- `50%` 表示限制为当前已分配 `vCPU` 总能力的一半。
- `100%` 表示允许使用当前已分配 `vCPU` 的全部能力。

## 生效逻辑

- 新建 / 导入 / 模板克隆 / 原生链式克隆：在生成 domain XML 时写入 `<cputune><period/><quota/></cputune>`。
- 编辑已有虚拟机：
  - 持久化配置通过 `virsh define` 更新。
  - 运行中的虚拟机会尝试通过 `virsh schedinfo --live` 同步运行态限速。
- 关闭限制时会移除当前面板写入的 `period/quota`，恢复为不限速。

## vCPU 变更兼容

如果虚拟机原本已经设置 CPU 限制，管理员在编辑页修改 `vCPU` 数量时，系统会自动按原百分比重算 `quota`，保持限制比例不变。

例如：

- 原配置：`2 vCPU`，`50%`
- 调整后：`4 vCPU`
- 系统会自动把实际 `quota` 重新计算为 `4 vCPU * 50%`

## 接口字段

以下管理员接口新增可选字段 `cpu_limit_percent`：

- `PUT /api/vm/:name`
- `POST /api/vm/create`
- `POST /api/vm/clone`
- `POST /api/vm/linked-clone`
- `POST /api/vm/import`
- 管理员轻量云待开通登记表单

字段规则：

- `0` 或不传：无限制
- `1-100`：按百分比限速

## 实现位置

- 前端表单：`web/src/components/VmForm.vue`
- 后端编辑：`server/handler/vm.go`
- CPU 限制逻辑：`server/service/vm_cpu_limit.go`
