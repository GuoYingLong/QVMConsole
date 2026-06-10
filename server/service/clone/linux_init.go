package clone

import (
	"fmt"
	"strings"
	"time"

	"kvm_console/utils"
)

func prepareLinuxCloneFirstBootIdentity(params *CloneParams, cloneDisk string) error {
	args := []string{
		"-a", cloneDisk,
		"--no-network",
	}
	for _, cmd := range buildLinuxFirstBootIdentityCommands(params.Hostname) {
		args = append(args, "--run-command", cmd)
	}
	args = append(args, "--quiet")

	result := utils.ExecCommandLongRunning("virt-customize", args...)
	if result.Error != nil {
		return fmt.Errorf("Linux 首次启动身份重置失败: %s", D.FirstNonEmpty(result.Stderr, result.Error.Error()))
	}
	return nil
}

func buildLinuxFirstBootIdentityCommands(hostname string) []string {
	return []string{
		"truncate -s 0 /etc/machine-id 2>/dev/null || rm -f /etc/machine-id",
		"rm -f /var/lib/dbus/machine-id 2>/dev/null || true",
		"rm -f /var/lib/dhcp/*.leases 2>/dev/null || true",
		"rm -f /var/lib/NetworkManager/*.lease 2>/dev/null || true",
		"rm -f /var/lib/systemd/network/*.lease 2>/dev/null || true",
		"rm -rf /run/systemd/netif/leases/* 2>/dev/null || true",
		"rm -rf /var/lib/cloud/instances/* /var/lib/cloud/instance 2>/dev/null || true",
		fmt.Sprintf("printf '%%s\\n' %s > /etc/hostname", utils.ShellSingleQuote(hostname)),
		buildLinuxHostsCommand(hostname),
	}
}

func buildLinuxHostsCommand(hostname string) string {
	return fmt.Sprintf(`TARGET_HOSTNAME=%s
if grep -q '^127\.0\.1\.1[[:space:]]' /etc/hosts; then
  sed -i "s/^127\.0\.1\.1[[:space:]].*/127.0.1.1\t${TARGET_HOSTNAME}/" /etc/hosts
else
  printf '127.0.1.1\t%%s\n' "$TARGET_HOSTNAME" >> /etc/hosts
fi`, utils.ShellSingleQuote(hostname))
}

// InitLinuxClone Linux 克隆初始化（SSH 设置 hostname/user/password/扩容）
func InitLinuxClone(params *CloneParams, ip string, progressFn func(int, string)) error {
	templateRootPass := params.TemplateRootPass
	if templateRootPass == "" {
		templateRootPass = "Qwert333"
	}
	templateUser := params.TemplateUser
	if templateUser == "" {
		templateUser = "xinyu"
	}

	progressFn(75, "等待 SSH 就绪...")
	time.Sleep(30 * time.Second)

	utils.ExecShell(fmt.Sprintf("ssh-keygen -f /root/.ssh/known_hosts -R %s 2>/dev/null", utils.ShellSingleQuote(ip)))

	sshUser := templateUser
	sshPass := templateRootPass
	sshReady := false

	for i := 0; i < 12; i++ {
		testResult := utils.ExecShell(fmt.Sprintf(
			"sshpass -p %s ssh -o StrictHostKeyChecking=no -o ConnectTimeout=3 -o UserKnownHostsFile=/dev/null %s 'echo ok' 2>/dev/null",
			utils.ShellSingleQuote(sshPass), utils.ShellSingleQuote(fmt.Sprintf("%s@%s", sshUser, ip))))
		if strings.TrimSpace(testResult.Stdout) == "ok" {
			sshReady = true
			break
		}
		time.Sleep(5 * time.Second)
	}

	if !sshReady {
		return fmt.Errorf("SSH 连接超时，Linux 初始化未执行")
	}

	progressFn(80, fmt.Sprintf("SSH 初始化 (用户: %s): hostname/用户/密码/磁盘扩容...", sshUser))

	var initCmds []string

	if !params.LinuxIdentityPrepared {
		initCmds = append(initCmds,
			"rm -f /etc/machine-id",
			"systemd-machine-id-setup",
			"rm -f /var/lib/dhcp/*.leases 2>/dev/null || true",
			"rm -f /var/lib/NetworkManager/*.lease 2>/dev/null || true",
			"rm -f /var/lib/systemd/network/*.lease 2>/dev/null || true",
			"rm -rf /run/systemd/netif/leases/* 2>/dev/null || true",
		)
	}

	initCmds = append(initCmds,
		fmt.Sprintf("hostnamectl set-hostname %s", utils.ShellSingleQuote(params.Hostname)),
		fmt.Sprintf("echo %s > /etc/hostname", utils.ShellSingleQuote(params.Hostname)),
		fmt.Sprintf("sed -i 's/127.0.1.1.*/127.0.1.1\\t%s/' /etc/hosts", params.Hostname),
	)

	if params.User != "" && params.User != templateUser {
		initCmds = append(initCmds,
			fmt.Sprintf("sed -i 's/^%s:/%s:/' /etc/passwd", templateUser, params.User),
			fmt.Sprintf("sed -i 's|/home/%s|/home/%s|' /etc/passwd", templateUser, params.User),
			fmt.Sprintf("sed -i 's/^%s:/%s:/' /etc/shadow", templateUser, params.User),
			fmt.Sprintf("sed -i 's/^%s:/%s:/' /etc/group", templateUser, params.User),
			fmt.Sprintf("sed -i 's/^%s:/%s:/' /etc/gshadow 2>/dev/null || true", templateUser, params.User),
			fmt.Sprintf("mv '/home/%s' '/home/%s' 2>/dev/null || true", templateUser, params.User),
			fmt.Sprintf("sed -i 's/%s/%s/g' /etc/sudoers.d/* 2>/dev/null || true", templateUser, params.User),
		)
	}

	if params.Password != "" {
		targetUser := params.User
		if targetUser == "" {
			targetUser = templateUser
		}
		initCmds = append(initCmds,
			fmt.Sprintf("printf '%%s:%%s\\n' %s %s | chpasswd", utils.ShellSingleQuote("root"), utils.ShellSingleQuote(params.Password)),
			fmt.Sprintf("printf '%%s:%%s\\n' %s %s | chpasswd", utils.ShellSingleQuote(targetUser), utils.ShellSingleQuote(params.Password)),
		)
	}

	initCmds = append(initCmds, buildLinuxDiskResizeScript())

	if !params.LinuxIdentityPrepared {
		initCmds = append(initCmds,
			"sleep 8",
			"systemctl restart systemd-networkd 2>/dev/null || netplan apply 2>/dev/null || systemctl restart NetworkManager 2>/dev/null || true",
		)
	}

	script := strings.Join(initCmds, "\n")

	var sshCmd string
	if sshUser == "root" {
		sshCmd = fmt.Sprintf(
			"sshpass -p %s ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null %s bash -s << 'INITEOF'\n%s\nINITEOF",
			utils.ShellSingleQuote(sshPass), utils.ShellSingleQuote("root@"+ip), script)
	} else {
		sshCmd = fmt.Sprintf(
			"sshpass -p %s ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null %s bash -s << 'INITEOF'\n"+
				"cat > /tmp/_clone_init.sh << 'SCRIPTEOF'\n%s\nSCRIPTEOF\n"+
				"if command -v sudo >/dev/null 2>&1 && printf '%%s\\n' %s | sudo -S -p '' true >/tmp/_clone_privilege_check.log 2>&1; then\n"+
				"  printf '%%s\\n' %s | sudo -S -p '' bash /tmp/_clone_init.sh > /tmp/_clone_init.log 2>&1\n"+
				"  INIT_STATUS=$?\n"+
				"else\n"+
				"  printf '%%s\\n' %s | su - root -c 'bash /tmp/_clone_init.sh > /tmp/_clone_init.log 2>&1'\n"+
				"  INIT_STATUS=$?\n"+
				"fi\n"+
				"if [ \"$INIT_STATUS\" -ne 0 ]; then\n"+
				"  echo '初始化脚本执行失败，日志如下:' >&2\n"+
				"  cat /tmp/_clone_init.log >&2 2>/dev/null || true\n"+
				"  exit \"$INIT_STATUS\"\n"+
				"fi\n"+
				"INITEOF",
			utils.ShellSingleQuote(sshPass), utils.ShellSingleQuote(fmt.Sprintf("%s@%s", sshUser, ip)), script,
			utils.ShellSingleQuote(sshPass), utils.ShellSingleQuote(sshPass), utils.ShellSingleQuote(sshPass))
	}

	result := utils.ExecShellWithTimeout(sshCmd, 5*time.Minute)
	if result.Error != nil {
		return fmt.Errorf("SSH 初始化执行失败: %s", D.FirstNonEmpty(result.Stderr, result.Error.Error()))
	}
	if params.Password != "" {
		targetUser := params.User
		if targetUser == "" {
			targetUser = templateUser
		}
		if err := waitForLinuxCloneCredential(targetUser, params.Password, ip, 90*time.Second); err != nil {
			return err
		}
	}
	progressFn(95, "SSH 初始化完成")
	return nil
}

func buildLinuxDiskResizeScript() string {
	return strings.TrimSpace(`
set -e

get_parent_disk() {
  local DEV="$1"
  local DEV_NAME
  local SYS_PATH
  local PARENT_NAME
  local PKNAME

  DEV_NAME=$(basename "$(readlink -f "$DEV" 2>/dev/null || echo "$DEV")")
  SYS_PATH=$(readlink -f "/sys/class/block/$DEV_NAME" 2>/dev/null || true)
  if [ -n "$SYS_PATH" ]; then
    PARENT_NAME=$(basename "$(dirname "$SYS_PATH")")
    if [ -n "$PARENT_NAME" ] && [ "$PARENT_NAME" != "block" ] && [ "$PARENT_NAME" != "$DEV_NAME" ]; then
      echo "/dev/$PARENT_NAME"
      return 0
    fi
  fi

  PKNAME=$(lsblk -no PKNAME "$DEV" 2>/dev/null | head -1 | tr -d ' ')
  if [ -n "$PKNAME" ] && [ "$PKNAME" != "$DEV_NAME" ]; then
    echo "/dev/$PKNAME"
    return 0
  fi
  if echo "$DEV" | grep -Eq 'p[0-9]+$'; then
    echo "$DEV" | sed -E 's/p[0-9]+$//'
  else
    echo "$DEV" | sed -E 's/[0-9]+$//'
  fi
}

get_partition_number() {
  local DEV="$1"
  local DEV_NAME
  local PART_NUM

  DEV_NAME=$(basename "$(readlink -f "$DEV" 2>/dev/null || echo "$DEV")")
  PART_NUM=$(cat "/sys/class/block/$DEV_NAME/partition" 2>/dev/null || true)
  if [ -n "$PART_NUM" ]; then
    echo "$PART_NUM"
    return 0
  fi

  PART_NUM=$(lsblk -no PARTN "$DEV" 2>/dev/null | head -1 | tr -d ' ')
  if echo "$PART_NUM" | grep -Eq '^[0-9]+$'; then
    echo "$PART_NUM"
    return 0
  fi
  echo "$DEV" | sed -E 's/^.*[^0-9]([0-9]+)$/\1/'
}

reread_partition_table() {
  local DISK="$1"
  partprobe "$DISK" 2>/dev/null || true
  partx -u "$DISK" 2>/dev/null || true
  blockdev --rereadpt "$DISK" 2>/dev/null || true
  udevadm settle 2>/dev/null || true
}

partition_has_grow_room() {
  local DISK="$1"
  local PART_DEV="$2"
  local PART_NAME
  local DISK_SECTORS
  local PART_START
  local PART_SECTORS
  local PART_END

  PART_NAME=$(basename "$(readlink -f "$PART_DEV" 2>/dev/null || echo "$PART_DEV")")
  DISK_SECTORS=$(blockdev --getsz "$DISK" 2>/dev/null || true)
  PART_START=$(cat "/sys/class/block/$PART_NAME/start" 2>/dev/null || true)
  PART_SECTORS=$(cat "/sys/class/block/$PART_NAME/size" 2>/dev/null || true)
  if [ -z "$DISK_SECTORS" ] || [ -z "$PART_START" ] || [ -z "$PART_SECTORS" ]; then
    return 0
  fi

  PART_END=$((PART_START + PART_SECTORS))
  [ $((DISK_SECTORS - PART_END)) -gt 2048 ]
}

grow_partition() {
  local PART_DEV="$1"
  local DISK
  local PART_NUM
  DISK=$(get_parent_disk "$PART_DEV")
  PART_NUM=$(get_partition_number "$PART_DEV")

  if [ -z "$DISK" ] || [ -z "$PART_NUM" ] || [ "$DISK" = "$PART_DEV" ]; then
    echo "无法识别根分区所在磁盘: $PART_DEV" >&2
    return 1
  fi

  if ! partition_has_grow_room "$DISK" "$PART_DEV"; then
    echo "分区已占满磁盘，跳过分区扩容: $PART_DEV"
    return 0
  fi

  if command -v growpart >/dev/null 2>&1; then
    if growpart "$DISK" "$PART_NUM"; then
      reread_partition_table "$DISK"
      return 0
    fi
    if ! partition_has_grow_room "$DISK" "$PART_DEV"; then
      echo "growpart 执行后分区已占满磁盘，继续后续扩容"
      return 0
    fi
    echo "growpart 扩容失败，尝试使用其他分区工具" >&2
  fi
	if command -v parted >/dev/null 2>&1; then
		parted -s "$DISK" resizepart "$PART_NUM" 100%
	elif command -v sfdisk >/dev/null 2>&1; then
		printf ', +\n' | sfdisk --no-reread -N "$PART_NUM" "$DISK"
	else
		echo "缺少分区扩容工具，请在模板内安装 cloud-guest-utils、parted 或 util-linux(sfdisk)" >&2
		return 1
	fi

	reread_partition_table "$DISK"
}

resize_filesystem() {
  local TARGET="$1"
  if command -v resize2fs >/dev/null 2>&1; then
    resize2fs "$TARGET" 2>/dev/null && return 0
  fi
  if command -v xfs_growfs >/dev/null 2>&1; then
    xfs_growfs / && return 0
  fi
  echo "缺少文件系统扩容工具，请在模板内安装 e2fsprogs 或 xfsprogs" >&2
  return 1
}

ROOT_DEV=$(findmnt -n -o SOURCE /)
if echo "$ROOT_DEV" | grep -q "mapper"; then
  VG_NAME=$(lvs --noheadings -o vg_name "$ROOT_DEV" 2>/dev/null | awk '{print $1}' | head -1)
  PV_DEV=$(pvs --noheadings -o pv_name,vg_name 2>/dev/null | awk -v vg="$VG_NAME" '$2 == vg {print $1; exit}')
  if [ -z "$PV_DEV" ]; then
    PV_DEV=$(pvs --noheadings -o pv_name 2>/dev/null | awk '{print $1}' | head -1)
  fi
  if [ -z "$PV_DEV" ]; then
    echo "未找到 LVM 物理卷，无法扩容根分区" >&2
    exit 1
  fi

  grow_partition "$PV_DEV"
  pvresize "$PV_DEV"

  FREE_EXTENTS=$(vgs --noheadings -o vg_free_count "$VG_NAME" 2>/dev/null | awk '{print $1}' | head -1)
  case "${FREE_EXTENTS:-0}" in
    ''|*[!0-9]*) FREE_EXTENTS=0 ;;
  esac
  if [ "$FREE_EXTENTS" -gt 0 ]; then
    if ! lvextend -r -l +100%FREE "$ROOT_DEV"; then
      echo "lvextend -r 失败，尝试分步扩容根逻辑卷"
      FREE_EXTENTS=$(vgs --noheadings -o vg_free_count "$VG_NAME" 2>/dev/null | awk '{print $1}' | head -1)
      case "${FREE_EXTENTS:-0}" in
        ''|*[!0-9]*) FREE_EXTENTS=0 ;;
      esac
      if [ "$FREE_EXTENTS" -gt 0 ]; then
        lvextend -l +100%FREE "$ROOT_DEV"
      fi
      resize_filesystem "$ROOT_DEV"
    fi
  else
    echo "VG 无可用空间，跳过 LV 扩容，仅检查文件系统"
    resize_filesystem "$ROOT_DEV"
  fi
else
  grow_partition "$ROOT_DEV"
  resize_filesystem "$ROOT_DEV"
fi
`)
}

func waitForLinuxCloneCredential(username, password, ip string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		result := utils.ExecShell(fmt.Sprintf(
			"sshpass -p %s ssh -o StrictHostKeyChecking=no -o ConnectTimeout=3 -o UserKnownHostsFile=/dev/null %s 'echo ok' 2>/dev/null",
			utils.ShellSingleQuote(password),
			utils.ShellSingleQuote(fmt.Sprintf("%s@%s", username, ip)),
		))
		if strings.TrimSpace(result.Stdout) == "ok" {
			return nil
		}
		time.Sleep(5 * time.Second)
	}
	return fmt.Errorf("Linux 初始化后无法使用新账号密码登录，请检查模板 sudo 权限或初始化日志")
}
