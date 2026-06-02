# Debian 模板开通初始化

## 背景

Debian 13 模板属于 Linux 模板分支，开通时复用现有 Linux SSH 初始化流程。克隆机会先等待系统获取 IP，再通过模板中记录的登录用户执行 SSH 初始化，完成主机名、登录用户、密码和系统盘扩容。

## 模板要求

- 模板分类固定选择 `Debian`，不允许录入自定义 Linux 分类。
- 模板登录用户建议使用普通用户，例如 `xiaozhu`，不要直接使用 `root`。Debian 默认可能禁止 root 通过 SSH 密码登录，即使 root 账户已启用也会导致初始化失败。
- 模板用户建议具备 sudo 权限，且模板内建议安装 `sudo`。
- 如果 Debian 模板未安装 `sudo` 或模板用户不在 sudo 组，系统会尝试使用同一模板密码通过 `su - root` 提权执行初始化；因此 root 本地账户密码必须与模板凭据一致且允许 `su`。
- 模板内需要安装 `openssh-server`，并启用 SSH 服务。
- 模板内建议安装 `cloud-guest-utils`、`e2fsprogs` 和 `lvm2`，用于 `growpart`、`resize2fs`、`pvresize`、`lvextend` 扩展系统盘。若模板缺少 `growpart`，系统会依次尝试 `parted` 和 `sfdisk` 回退扩展分区，但生产模板仍建议预装 `cloud-guest-utils`。

## UEFI 与安全引导

UEFI 模板制作时会保存源虚拟机的 NVRAM sidecar 文件。后续从模板克隆 UEFI 虚拟机时，系统会优先复制该 NVRAM 到新虚拟机，避免首次启动时因为全新 OVMF 变量文件缺少启动项而进入 `Boot Option Restoration`。

如果旧模板没有 NVRAM sidecar，仍会沿用原有 UEFI 启动方式；这类模板首次启动时可能短暂出现固件恢复启动项页面，只要磁盘内存在 `EFI/BOOT/BOOTX64.EFI` fallback，系统通常仍可继续启动。

## 故障排查

- `未获取到虚拟机 IP，Linux 初始化无法执行`：优先确认系统是否已进入 Debian、VPC/OVS 邻居表是否有 VM MAC 对应 IP。Debian 首启和 UEFI 恢复启动项可能较慢，当前等待时间已延长到 180 秒。
- `SSH 连接超时，Linux 初始化未执行`：检查模板元数据中的 `template_user` 和 `root_password` 是否对应一个允许 SSH 密码登录的模板用户。
- `Linux 初始化后无法使用新账号密码登录`：检查 `/tmp/_clone_init.log` 和 `/tmp/_clone_privilege_check.log`。如果模板没有 sudo，需要确认模板用户可通过 `su - root` 使用模板密码提权。
- `磁盘没有扩容成功`：检查克隆机内 `/tmp/_clone_init.log`。新流程会优先通过 `/sys/class/block` 识别根分区父磁盘，并在分区扩容、LVM 扩容或文件系统扩容失败时让开通任务失败，不再因为新账号密码可登录而掩盖初始化错误。若模板使用 LVM，确认模板内存在 `lvm2`；若使用 ext4，确认存在 `e2fsprogs`；若使用 XFS，确认存在 `xfsprogs`。
