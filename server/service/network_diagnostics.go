package service

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"kvm_console/config"
	"kvm_console/utils"
)

const maxCaptureSummaryLines = 300

var captureInterfaceRe = regexp.MustCompile(`^[A-Za-z0-9_.:-]+$`)

type NetworkDiagnosticFilter struct {
	Protocol   string `json:"protocol"`
	SourceIP   string `json:"source_ip"`
	DestIP     string `json:"dest_ip"`
	Port       int    `json:"port"`
	SourcePort int    `json:"source_port"`
	DestPort   int    `json:"dest_port"`
}

type NetworkCaptureRequest struct {
	InterfaceName   string                  `json:"interface_name"`
	Filter          NetworkDiagnosticFilter `json:"filter"`
	DurationSeconds int                     `json:"duration_seconds"`
	MaxMB           int                     `json:"max_mb"`
	MaxPackets      int                     `json:"max_packets"`
}

type NetworkCaptureParams struct {
	VMName string `json:"vm_name"`
	NetworkCaptureRequest
}

type NetworkDiagnosticTemplate struct {
	Key         string                  `json:"key"`
	Name        string                  `json:"name"`
	Description string                  `json:"description"`
	Filter      NetworkDiagnosticFilter `json:"filter"`
}

type VMNetworkDiagnostics struct {
	VMName           string                      `json:"vm_name"`
	State            string                      `json:"state"`
	Interfaces       []VMNetworkInterface        `json:"interfaces"`
	Neighbors        []string                    `json:"neighbors"`
	Templates        []NetworkDiagnosticTemplate `json:"templates"`
	PortForwards     []PortForwardRule           `json:"port_forwards"`
	DefaultInterface string                      `json:"default_interface"`
	DefaultIP        string                      `json:"default_ip"`
	Issues           []string                    `json:"issues"`
}

type NetworkCaptureSession struct {
	TaskID          uint                    `json:"task_id"`
	VMName          string                  `json:"vm_name"`
	InterfaceName   string                  `json:"interface_name"`
	Filter          NetworkDiagnosticFilter `json:"filter"`
	BPF             string                  `json:"bpf"`
	Status          string                  `json:"status"`
	Message         string                  `json:"message"`
	FileName        string                  `json:"file_name"`
	DownloadPath    string                  `json:"download_path"`
	FileSize        int64                   `json:"file_size"`
	DurationSeconds int                     `json:"duration_seconds"`
	MaxMB           int                     `json:"max_mb"`
	MaxPackets      int                     `json:"max_packets"`
	SummaryLines    []string                `json:"summary_lines"`
	StartedAt       time.Time               `json:"started_at"`
	UpdatedAt       time.Time               `json:"updated_at"`
	FinishedAt      *time.Time              `json:"finished_at,omitempty"`
}

var (
	captureSessions   = make(map[uint]*NetworkCaptureSession)
	captureSessionsMu sync.RWMutex
)

func ParseNetworkCaptureParams(raw string) (NetworkCaptureParams, error) {
	var params NetworkCaptureParams
	if err := json.Unmarshal([]byte(raw), &params); err != nil {
		return params, err
	}
	params.VMName = strings.TrimSpace(params.VMName)
	if params.VMName == "" {
		return params, fmt.Errorf("虚拟机名称不能为空")
	}
	return params, nil
}

func GetVMNetworkDiagnostics(vmName string) (*VMNetworkDiagnostics, error) {
	status, err := GetVMNetworkRuntimeStatus(vmName)
	if err != nil {
		return nil, err
	}
	diag := &VMNetworkDiagnostics{
		VMName:     status.VMName,
		State:      status.State,
		Interfaces: status.Interfaces,
		Issues:     append([]string{}, status.Issues...),
	}
	for _, iface := range status.Interfaces {
		if isUsableCaptureInterface(iface) {
			diag.DefaultInterface = iface.Target
			diag.DefaultIP = iface.IP
			break
		}
	}
	if diag.DefaultInterface != "" {
		diag.Neighbors = readNetworkNeighbors(diag.DefaultInterface)
	}
	diag.PortForwards = portForwardsForVMInterfaces(status.Interfaces)
	diag.Templates = buildNetworkDiagnosticTemplates(diag.DefaultIP, diag.PortForwards)
	if diag.DefaultInterface == "" {
		diag.Issues = append(diag.Issues, "未找到可抓包的运行中 vnet/tap 接口")
	}
	return diag, nil
}

func InitNetworkCaptureSession(taskID uint, vmName string, req NetworkCaptureRequest, createdBy string) {
	now := time.Now()
	pruneOldCaptureSessions(now.Add(-24 * time.Hour))
	deletePreviousCaptureFile(taskID)
	session := &NetworkCaptureSession{
		TaskID:          taskID,
		VMName:          strings.TrimSpace(vmName),
		InterfaceName:   strings.TrimSpace(req.InterfaceName),
		Filter:          req.Filter,
		Status:          "pending",
		Message:         "抓包任务已提交",
		DurationSeconds: req.DurationSeconds,
		MaxMB:           req.MaxMB,
		MaxPackets:      req.MaxPackets,
		StartedAt:       now,
		UpdatedAt:       now,
	}
	captureSessionsMu.Lock()
	captureSessions[taskID] = session
	captureSessionsMu.Unlock()
}

func GetNetworkCaptureSession(taskID uint) (*NetworkCaptureSession, bool) {
	captureSessionsMu.RLock()
	defer captureSessionsMu.RUnlock()
	session, ok := captureSessions[taskID]
	if !ok {
		return nil, false
	}
	cp := *session
	cp.SummaryLines = append([]string{}, session.SummaryLines...)
	if cp.FileName != "" {
		cp.FileSize = captureFileSize(filepath.Join(networkCaptureDir(), cp.FileName))
	}
	return &cp, true
}

func DeleteNetworkCaptureFile(taskID uint) error {
	session, ok := GetNetworkCaptureSession(taskID)
	if !ok {
		return fmt.Errorf("抓包任务不存在或已过期")
	}
	if session.Status == "running" {
		return fmt.Errorf("抓包仍在运行，请先取消或等待完成后再删除")
	}
	if strings.TrimSpace(session.FileName) == "" {
		return nil
	}
	filePath, _, err := NetworkCaptureFilePath(taskID)
	if err != nil {
		updateNetworkCaptureSession(taskID, func(s *NetworkCaptureSession) {
			s.FileName = ""
			s.DownloadPath = ""
			s.FileSize = 0
			s.Message = "pcap 文件已不存在"
		})
		return nil
	}
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("删除 pcap 文件失败: %w", err)
	}
	updateNetworkCaptureSession(taskID, func(s *NetworkCaptureSession) {
		s.FileName = ""
		s.DownloadPath = ""
		s.FileSize = 0
		s.Message = "pcap 文件已删除"
	})
	return nil
}

func NetworkCaptureFilePath(taskID uint) (string, string, error) {
	session, ok := GetNetworkCaptureSession(taskID)
	if !ok || session.FileName == "" {
		return "", "", fmt.Errorf("抓包文件不存在或任务尚未完成")
	}
	fileName := filepath.Base(session.FileName)
	filePath := filepath.Join(networkCaptureDir(), fileName)
	absDir, _ := filepath.Abs(networkCaptureDir())
	absFile, _ := filepath.Abs(filePath)
	if absDir != "" && !strings.HasPrefix(absFile, absDir+string(os.PathSeparator)) && absFile != absDir {
		return "", "", fmt.Errorf("抓包文件路径异常")
	}
	if _, err := os.Stat(filePath); err != nil {
		return "", "", fmt.Errorf("抓包文件不存在或已过期")
	}
	return filePath, fileName, nil
}

func ExecuteNetworkCapture(ctx context.Context, taskID uint, params NetworkCaptureParams, progress func(int, string)) (string, error) {
	if _, err := exec.LookPath("tcpdump"); err != nil {
		captureErr := fmt.Errorf("未检测到 tcpdump，请先安装 tcpdump 后再执行抓包")
		failNetworkCaptureSession(taskID, captureErr)
		return "", captureErr
	}
	req, iface, bpf, err := normalizeNetworkCaptureRequest(params.VMName, params.NetworkCaptureRequest)
	if err != nil {
		failNetworkCaptureSession(taskID, err)
		return "", err
	}
	updateNetworkCaptureSession(taskID, func(session *NetworkCaptureSession) {
		session.VMName = params.VMName
		session.InterfaceName = iface
		session.Filter = req.Filter
		session.BPF = bpf
		session.DurationSeconds = req.DurationSeconds
		session.MaxMB = req.MaxMB
		session.MaxPackets = req.MaxPackets
		session.Status = "running"
		session.Message = "正在抓包..."
	})
	if progress != nil {
		progress(10, "正在准备抓包环境...")
	}
	if err := os.MkdirAll(networkCaptureDir(), 0o750); err != nil {
		captureErr := fmt.Errorf("创建抓包目录失败: %w", err)
		failNetworkCaptureSession(taskID, captureErr)
		return "", captureErr
	}
	fileName := fmt.Sprintf("capture-%d-%s-%s.pcap", taskID, sanitizeFilePart(params.VMName), time.Now().Format("20060102-150405"))
	filePath := filepath.Join(networkCaptureDir(), fileName)
	updateNetworkCaptureSession(taskID, func(session *NetworkCaptureSession) {
		session.FileName = fileName
		session.DownloadPath = fmt.Sprintf("/api/network/captures/%d/download", taskID)
	})

	captureCtx, cancel := context.WithTimeout(ctx, time.Duration(req.DurationSeconds)*time.Second)
	defer cancel()
	errCh := make(chan error, 2)
	go func() {
		errCh <- runTcpdumpToFile(captureCtx, iface, filePath, req.MaxPackets, bpf)
	}()
	go func() {
		errCh <- runTcpdumpSummary(captureCtx, taskID, iface, req.MaxPackets, bpf)
	}()
	go monitorCaptureFileSize(captureCtx, cancel, taskID, filePath, int64(req.MaxMB)*1024*1024)

	if progress != nil {
		progress(30, "抓包进行中...")
	}
	var firstErr error
	for i := 0; i < 2; i++ {
		err := <-errCh
		if err != nil && firstErr == nil && captureCtx.Err() == nil {
			firstErr = err
			cancel()
		}
	}
	if progress != nil {
		progress(90, "正在整理抓包结果...")
	}
	now := time.Now()
	fileSize := captureFileSize(filePath)
	status := "success"
	message := "抓包完成"
	if ctx.Err() == context.Canceled {
		status = "canceled"
		message = "抓包任务已取消"
		firstErr = ctx.Err()
	} else if firstErr != nil {
		status = "failed"
		message = firstErr.Error()
	}
	updateNetworkCaptureSession(taskID, func(session *NetworkCaptureSession) {
		session.Status = status
		session.Message = message
		session.FileSize = fileSize
		session.FinishedAt = &now
	})
	result := map[string]interface{}{
		"task_id":       taskID,
		"vm_name":       params.VMName,
		"interface":     iface,
		"bpf":           bpf,
		"file_name":     fileName,
		"download_path": fmt.Sprintf("/api/network/captures/%d/download", taskID),
		"file_size":     fileSize,
	}
	data, _ := json.Marshal(result)
	if firstErr != nil {
		return string(data), firstErr
	}
	if progress != nil {
		progress(100, "抓包完成")
	}
	return string(data), nil
}

func normalizeNetworkCaptureRequest(vmName string, req NetworkCaptureRequest) (NetworkCaptureRequest, string, string, error) {
	status, err := GetVMNetworkRuntimeStatus(vmName)
	if err != nil {
		return req, "", "", err
	}
	if strings.TrimSpace(status.State) != "running" {
		return req, "", "", fmt.Errorf("虚拟机未运行，无法对运行态接口抓包")
	}
	selected := strings.TrimSpace(req.InterfaceName)
	var matched *VMNetworkInterface
	for i := range status.Interfaces {
		if !isUsableCaptureInterface(status.Interfaces[i]) {
			continue
		}
		if selected == "" || status.Interfaces[i].Target == selected {
			matched = &status.Interfaces[i]
			selected = status.Interfaces[i].Target
			break
		}
	}
	if matched == nil {
		return req, "", "", fmt.Errorf("未找到可抓包的 VM 运行态接口")
	}
	if !captureInterfaceRe.MatchString(selected) {
		return req, "", "", fmt.Errorf("接口名称不合法")
	}
	req.DurationSeconds = clampInt(req.DurationSeconds, captureDefaultSeconds(), captureMaxSeconds())
	req.MaxMB = clampInt(req.MaxMB, captureMaxMB(), captureMaxMB())
	req.MaxPackets = clampInt(req.MaxPackets, captureMaxPackets(), captureMaxPackets())
	bpf, err := BuildNetworkCaptureBPF(req.Filter)
	if err != nil {
		return req, "", "", err
	}
	return req, selected, bpf, nil
}

func BuildNetworkCaptureBPF(filter NetworkDiagnosticFilter) (string, error) {
	var parts []string
	protocol := strings.ToLower(strings.TrimSpace(filter.Protocol))
	switch protocol {
	case "", "any", "all":
	case "tcp", "udp", "icmp", "arp":
		parts = append(parts, protocol)
	case "dhcp":
		parts = append(parts, "(udp and (port 67 or port 68))")
	case "dns":
		parts = append(parts, "port 53")
	default:
		return "", fmt.Errorf("不支持的协议过滤条件")
	}
	if (protocol == "arp" || protocol == "icmp") && (filter.Port > 0 || filter.SourcePort > 0 || filter.DestPort > 0) {
		return "", fmt.Errorf("%s 协议不能同时指定端口过滤", strings.ToUpper(protocol))
	}
	if src := strings.TrimSpace(filter.SourceIP); src != "" {
		if net.ParseIP(src) == nil {
			return "", fmt.Errorf("源 IP 格式不正确")
		}
		parts = append(parts, "src host "+src)
	}
	if dst := strings.TrimSpace(filter.DestIP); dst != "" {
		if net.ParseIP(dst) == nil {
			return "", fmt.Errorf("目标 IP 格式不正确")
		}
		parts = append(parts, "dst host "+dst)
	}
	if filter.Port > 0 {
		if !validPort(filter.Port) {
			return "", fmt.Errorf("端口范围必须为 1-65535")
		}
		parts = append(parts, "port "+strconv.Itoa(filter.Port))
	}
	if filter.SourcePort > 0 {
		if !validPort(filter.SourcePort) {
			return "", fmt.Errorf("源端口范围必须为 1-65535")
		}
		parts = append(parts, "src port "+strconv.Itoa(filter.SourcePort))
	}
	if filter.DestPort > 0 {
		if !validPort(filter.DestPort) {
			return "", fmt.Errorf("目标端口范围必须为 1-65535")
		}
		parts = append(parts, "dst port "+strconv.Itoa(filter.DestPort))
	}
	return strings.Join(parts, " and "), nil
}

func runTcpdumpToFile(ctx context.Context, iface, filePath string, packets int, bpf string) error {
	args := []string{"-i", iface, "-nn", "-s", "0", "-U", "-w", filePath, "-c", strconv.Itoa(packets)}
	if bpf != "" {
		args = append(args, bpf)
	}
	cmd := exec.CommandContext(ctx, "tcpdump", args...)
	output, err := cmd.CombinedOutput()
	if err != nil && ctx.Err() == nil {
		return fmt.Errorf("tcpdump 写入 pcap 失败: %s", strings.TrimSpace(string(output)))
	}
	return nil
}

func runTcpdumpSummary(ctx context.Context, taskID uint, iface string, packets int, bpf string) error {
	args := []string{"-i", iface, "-nn", "-l", "-tttt", "-s", "160", "-c", strconv.Itoa(packets)}
	if bpf != "" {
		args = append(args, bpf)
	}
	cmd := exec.CommandContext(ctx, "tcpdump", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动 tcpdump 摘要失败: %w", err)
	}
	var wg sync.WaitGroup
	readPipe := func(scanner *bufio.Scanner) {
		defer wg.Done()
		scanner.Buffer(make([]byte, 1024), 1024*1024)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line != "" {
				appendCaptureSummaryLine(taskID, line)
			}
		}
	}
	wg.Add(2)
	go readPipe(bufio.NewScanner(stdout))
	go readPipe(bufio.NewScanner(stderr))
	err = cmd.Wait()
	wg.Wait()
	if err != nil && ctx.Err() == nil {
		return fmt.Errorf("tcpdump 摘要输出失败: %w", err)
	}
	return nil
}

func monitorCaptureFileSize(ctx context.Context, cancel context.CancelFunc, taskID uint, filePath string, maxBytes int64) {
	if maxBytes <= 0 {
		return
	}
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			size := captureFileSize(filePath)
			updateNetworkCaptureSession(taskID, func(session *NetworkCaptureSession) {
				session.FileSize = size
			})
			if size >= maxBytes {
				appendCaptureSummaryLine(taskID, "抓包文件达到大小上限，已停止抓包")
				cancel()
				return
			}
		}
	}
}

func updateNetworkCaptureSession(taskID uint, fn func(*NetworkCaptureSession)) {
	now := time.Now()
	captureSessionsMu.Lock()
	defer captureSessionsMu.Unlock()
	session, ok := captureSessions[taskID]
	if !ok {
		session = &NetworkCaptureSession{TaskID: taskID, StartedAt: now}
		captureSessions[taskID] = session
	}
	fn(session)
	session.UpdatedAt = now
}

func deletePreviousCaptureFile(currentTaskID uint) {
	var previousTaskID uint
	var previousFileName string
	var previousUpdatedAt time.Time
	captureSessionsMu.RLock()
	for id, session := range captureSessions {
		if id == currentTaskID || session.Status == "running" || strings.TrimSpace(session.FileName) == "" {
			continue
		}
		if previousTaskID == 0 || session.UpdatedAt.After(previousUpdatedAt) {
			previousTaskID = id
			previousFileName = session.FileName
			previousUpdatedAt = session.UpdatedAt
		}
	}
	captureSessionsMu.RUnlock()
	if previousTaskID == 0 {
		return
	}
	fileName := filepath.Base(previousFileName)
	if fileName == "" || fileName == "." {
		return
	}
	_ = os.Remove(filepath.Join(networkCaptureDir(), fileName))
	updateNetworkCaptureSession(previousTaskID, func(session *NetworkCaptureSession) {
		session.FileName = ""
		session.DownloadPath = ""
		session.FileSize = 0
		session.Message = "pcap 文件已在新抓包开始前自动删除"
	})
}

func failNetworkCaptureSession(taskID uint, err error) {
	now := time.Now()
	updateNetworkCaptureSession(taskID, func(session *NetworkCaptureSession) {
		session.Status = "failed"
		if err != nil {
			session.Message = err.Error()
		} else {
			session.Message = "抓包失败"
		}
		session.FinishedAt = &now
	})
}

func appendCaptureSummaryLine(taskID uint, line string) {
	updateNetworkCaptureSession(taskID, func(session *NetworkCaptureSession) {
		session.SummaryLines = append(session.SummaryLines, line)
		if len(session.SummaryLines) > maxCaptureSummaryLines {
			session.SummaryLines = session.SummaryLines[len(session.SummaryLines)-maxCaptureSummaryLines:]
		}
	})
}

func pruneOldCaptureSessions(cutoff time.Time) {
	captureSessionsMu.Lock()
	defer captureSessionsMu.Unlock()
	for id, session := range captureSessions {
		if !session.UpdatedAt.IsZero() && session.UpdatedAt.Before(cutoff) && session.Status != "running" {
			delete(captureSessions, id)
		}
	}
}

func isUsableCaptureInterface(iface VMNetworkInterface) bool {
	target := strings.TrimSpace(iface.Target)
	if target == "" || target == "-" {
		return false
	}
	if !(strings.HasPrefix(target, "vnet") || strings.HasPrefix(target, "tap")) {
		return false
	}
	return iface.OFPort != "" && iface.OFPort != "-1"
}

func readNetworkNeighbors(iface string) []string {
	result := utils.ExecCommand("ip", "neigh", "show", "dev", iface)
	if result.Error != nil || strings.TrimSpace(result.Stdout) == "" {
		return []string{}
	}
	var lines []string
	for _, line := range strings.Split(result.Stdout, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

func portForwardsForVMInterfaces(interfaces []VMNetworkInterface) []PortForwardRule {
	ips := make(map[string]bool)
	for _, iface := range interfaces {
		if iface.IP != "" {
			ips[iface.IP] = true
		}
	}
	rules, err := listLivePortForwardsFromIPTables()
	if err != nil {
		return []PortForwardRule{}
	}
	var result []PortForwardRule
	for _, rule := range rules {
		if ips[rule.DestIP] {
			result = append(result, rule)
		}
	}
	return result
}

func buildNetworkDiagnosticTemplates(defaultIP string, forwards []PortForwardRule) []NetworkDiagnosticTemplate {
	templates := []NetworkDiagnosticTemplate{
		{Key: "arp", Name: "ARP", Description: "检查 ARP 请求与应答", Filter: NetworkDiagnosticFilter{Protocol: "arp"}},
		{Key: "dhcp", Name: "DHCP", Description: "检查 DHCP 获取地址过程", Filter: NetworkDiagnosticFilter{Protocol: "dhcp"}},
		{Key: "dns", Name: "DNS", Description: "检查 DNS 查询与响应", Filter: NetworkDiagnosticFilter{Protocol: "dns"}},
	}
	if defaultIP != "" {
		templates = append(templates, NetworkDiagnosticTemplate{
			Key:         "vm_ip",
			Name:        "当前 VM IP",
			Description: "只查看当前 VM IP 的流量",
			Filter:      NetworkDiagnosticFilter{SourceIP: defaultIP},
		})
	}
	for _, rule := range forwards {
		port, _ := strconv.Atoi(rule.DestPort)
		if port <= 0 {
			continue
		}
		protocol := strings.ToLower(rule.Protocol)
		if protocol != "tcp" && protocol != "udp" {
			protocol = "any"
		}
		templates = append(templates, NetworkDiagnosticTemplate{
			Key:         "pf_" + rule.StableKey(),
			Name:        fmt.Sprintf("端口转发 %s/%s", rule.DestPort, strings.ToUpper(rule.Protocol)),
			Description: fmt.Sprintf("检查宿主机端口 %s 到 VM %s:%s 的入站流量", rule.HostPort, rule.DestIP, rule.DestPort),
			Filter:      NetworkDiagnosticFilter{Protocol: protocol, DestIP: rule.DestIP, DestPort: port},
		})
	}
	return templates
}

func networkCaptureDir() string {
	if config.GlobalConfig == nil || strings.TrimSpace(config.GlobalConfig.NetworkCaptureDir) == "" {
		return "/var/lib/kvm-console/captures"
	}
	return config.GlobalConfig.NetworkCaptureDir
}

func captureDefaultSeconds() int {
	if config.GlobalConfig == nil || config.GlobalConfig.NetworkCaptureDefaultSeconds <= 0 {
		return 30
	}
	return config.GlobalConfig.NetworkCaptureDefaultSeconds
}

func captureMaxSeconds() int {
	if config.GlobalConfig == nil || config.GlobalConfig.NetworkCaptureMaxSeconds <= 0 {
		return 120
	}
	return config.GlobalConfig.NetworkCaptureMaxSeconds
}

func captureMaxMB() int {
	if config.GlobalConfig == nil || config.GlobalConfig.NetworkCaptureMaxMB <= 0 {
		return 64
	}
	return config.GlobalConfig.NetworkCaptureMaxMB
}

func captureMaxPackets() int {
	if config.GlobalConfig == nil || config.GlobalConfig.NetworkCaptureMaxPackets <= 0 {
		return 5000
	}
	return config.GlobalConfig.NetworkCaptureMaxPackets
}

func clampInt(value, defaultValue, maxValue int) int {
	if value <= 0 {
		value = defaultValue
	}
	if maxValue > 0 && value > maxValue {
		value = maxValue
	}
	return value
}

func validPort(port int) bool {
	return port >= 1 && port <= 65535
}

func captureFileSize(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.Size()
}

func sanitizeFilePart(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "vm"
	}
	var builder strings.Builder
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			builder.WriteRune(r)
		} else {
			builder.WriteByte('_')
		}
	}
	result := builder.String()
	if result == "" {
		return "vm"
	}
	return result
}
