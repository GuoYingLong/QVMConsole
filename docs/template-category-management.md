# 模板二级分类

## 功能说明

模板现在保留 Linux 和 Windows 二级分类字段，用于在模板制作、模板管理和“从模板创建虚拟机”入口中按系统分支展示模板。

- 顶级类型仍然沿用原有 `linux / windows / fnos / other`
- `linux` 支持 `Ubuntu`、`Debian`
- `windows` 支持 `WindowsServer2022`、`Windows10`、`WindowsServer2012R2`
- 当前界面会按 `Linux / Ubuntu`、`Linux / Debian`、`Windows / WindowsServer2022`、`Windows / Windows10`、`Windows / WindowsServer2012R2` 分组展示

## 默认行为

- 新制作的 Linux 模板会自动归入 `Ubuntu`
- 历史 Linux 模板如果没有分类或分类不在固定选项中，会按 `Ubuntu` 处理
- 新制作的 Windows 模板会自动归入 `WindowsServer2022`
- 历史 Windows 模板如果没有分类，会按模板文件名归类：名称包含 `Windows10` / `Win10` 时归入 `Windows10`，名称包含 `WindowsServer2012R2` / `Server2012R2` / `Win2k12R2` 时归入 `WindowsServer2012R2`，其余 Windows 模板归入 `WindowsServer2022`
- FnOS、Other 模板不使用该字段

## 管理方式

当前模板制作和模板管理页的分类字段只提供固定选项，不允许录入自定义分类。Debian 13 模板属于 `linux` 类型，开通时复用 Ubuntu 所在的 Linux SSH 初始化流程：克隆后等待 SSH 就绪，再设置主机名、登录用户、密码并扩展系统盘。

Windows 模板仍属于 `windows` 类型，二级分类只影响模板管理和克隆入口展示，不改变 Windows 初始化流程。`WindowsServer2012R2.qcow2` 应归入 `WindowsServer2012R2`，并可在模板发布设置中保存 BIOS、SATA、e1000e、VMVGA 等默认硬件配置；从模板创建时后端会按模板元数据使用对应启动方式和磁盘总线。

## 兼容性

- 旧模板无需迁移数据库，分类直接保存在模板 `.meta.json` 中
- 不影响模板链路、默认硬件配置、导入导出和克隆流程
- 对 FnOS、Other 模板不会新增额外必填项
