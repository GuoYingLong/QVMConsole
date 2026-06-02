# 模板链路删除

## 功能说明

模板删除支持两种模式：

- `cascade`：按节点子树执行。删除某个模板节点时，系统会同时删除该节点下的所有子节点。
- `promote_children`：仅删除当前节点，将当前节点的直接子模板提升到上级节点，并将当前节点直接创建的 VM 重定向到上级模板。
- `promote_children_hot`：热删除当前节点并提升子节点，运行中的关联 VM 通过 libvirt block job 在线处理。

删除前会获取预览信息：

- 将删除的模板节点
- 每个节点的用户侧显示文本
- 直接虚拟机数量和子树虚拟机总数
- 所有关联虚拟机
- 提升模式下将被提升的直接子模板
- 提升模式下需要安全 rebase 的直接关联 VM

级联删除确认后，系统先删除关联虚拟机，再删除模板磁盘和 `.meta.json`。

提升模式确认后，系统会：

1. 要求当前节点子树下所有关联 VM 均处于关机状态。
2. 使用安全 `qemu-img rebase` 将直接子模板的 backing 从当前节点改为上级模板。
3. 使用安全 `qemu-img rebase` 将当前节点直接关联 VM 的 backing 从当前节点改为上级模板。
4. 更新直接子模板的 `parent_node_id`、哈希和文件大小。
5. 更新直接关联 VM 的模板来源元数据。
6. 仅删除当前模板节点的磁盘和 `.meta.json`。

提升模式不会把子模板或 VM 磁盘转换为独立硬盘，处理后仍保持链式结构。

热提升模式会额外处理运行中的 VM：

- 当前节点直接创建的运行中 VM 使用 `virsh blockpull --base <上级模板>` 在线把当前节点差异拉入 VM overlay，再让 VM overlay 继承上级模板。
- 直接子模板会先复制到临时文件并安全 rebase 到上级模板，再原子替换原模板路径。
- 正在使用直接子模板或其后代模板的运行中 VM，会通过浅层 `virsh blockcopy --pivot --shallow --reuse-external` 切换到新子模板 backing。
- 如果热处理任一步失败，任务会停止并保留当前模板节点，不继续删除中间节点。

## 接口说明

### 删除预览

```http
GET /api/template/:name/delete-preview
```

返回内容包含：

- `templates`：将删除的模板节点列表
- `related_vms`：将联动删除的虚拟机列表
- `parent_template`：提升模式下的新上级模板
- `promoted_templates`：提升模式下会重新挂到上级模板的直接子模板
- `rebased_vms`：提升模式下会重定向 backing 的直接关联 VM
- `can_promote` / `promote_blockers`：当前是否满足提升模式条件和阻断原因
- `can_promote_hot` / `promote_hot_blockers`：当前是否满足热提升模式条件和阻断原因

### 执行删除

```http
DELETE /api/template/:name
Content-Type: application/json
```

请求体：

```json
{
  "delete_mode": "cascade",
  "delete_vms": true,
  "expected_vms": ["demo-vm-01"]
}
```

后端会在任务执行前再次比对 `expected_vms`。如果关联虚拟机列表发生变化，会拒绝删除并要求重新确认。

提升模式请求示例：

```json
{
  "delete_mode": "promote_children",
  "delete_vms": false,
  "expected_vms": ["demo-vm-01"]
}
```

热提升模式请求示例：

```json
{
  "delete_mode": "promote_children_hot",
  "delete_vms": false,
  "expected_vms": ["demo-vm-01"]
}
```
