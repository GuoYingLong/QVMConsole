package service

import (
	"encoding/xml"
	"fmt"
	"os"
	"regexp"
	"strings"

	"kvm_console/utils"
)

var vmXMLTempNameSanitizer = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

type vmXMLDomainEnvelope struct {
	XMLName xml.Name `xml:"domain"`
	Name    string   `xml:"name"`
}

func normalizeDomainXMLForEdit(xmlContent string) string {
	return strings.ReplaceAll(xmlContent, "\r\n", "\n")
}

// ValidateVMInactiveDomainXML 校验用于编辑的 domain XML。
func ValidateVMInactiveDomainXML(name, xmlContent string) error {
	trimmed := strings.TrimSpace(xmlContent)
	if trimmed == "" {
		return fmt.Errorf("XML 内容不能为空")
	}

	decoder := xml.NewDecoder(strings.NewReader(trimmed))
	decoder.Strict = true

	var domain vmXMLDomainEnvelope
	if err := decoder.Decode(&domain); err != nil {
		return fmt.Errorf("XML 格式不合法: %w", err)
	}
	if domain.XMLName.Local != "domain" {
		return fmt.Errorf("XML 根节点必须是 domain")
	}

	xmlName := strings.TrimSpace(domain.Name)
	if xmlName == "" {
		return fmt.Errorf("XML 中缺少虚拟机名称")
	}
	if xmlName != strings.TrimSpace(name) {
		return fmt.Errorf("不支持通过 XML 编辑修改虚拟机名称")
	}

	return nil
}

func buildDomainXMLTempPattern(name string) string {
	safeName := vmXMLTempNameSanitizer.ReplaceAllString(strings.TrimSpace(name), "_")
	if safeName == "" {
		safeName = "vm"
	}
	return fmt.Sprintf("_domain-xml-%s-*.xml", safeName)
}

// GetVMInactiveDomainXML 获取虚拟机持久化配置对应的 domain XML。
func GetVMInactiveDomainXML(name string) (string, error) {
	xmlResult := utils.ExecCommand("virsh", "dumpxml", name, "--inactive")
	if xmlResult.Error != nil {
		return "", fmt.Errorf("获取虚拟机 XML 失败: %s", xmlResult.Stderr)
	}
	return normalizeDomainXMLForEdit(xmlResult.Stdout), nil
}

// SetVMInactiveDomainXML 写入虚拟机持久化配置对应的 domain XML。
func SetVMInactiveDomainXML(name, xmlContent string) error {
	normalized := normalizeDomainXMLForEdit(xmlContent)
	if err := ValidateVMInactiveDomainXML(name, normalized); err != nil {
		return err
	}

	tmpFile, err := os.CreateTemp("/tmp", buildDomainXMLTempPattern(name))
	if err != nil {
		return fmt.Errorf("创建临时 XML 文件失败: %w", err)
	}
	tmpPath := tmpFile.Name()
	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("关闭临时 XML 文件失败: %w", err)
	}
	defer os.Remove(tmpPath)

	if err := os.WriteFile(tmpPath, []byte(normalized), 0600); err != nil {
		return fmt.Errorf("写入临时 XML 文件失败: %w", err)
	}

	defineResult := utils.ExecCommand("virsh", "define", tmpPath)
	if defineResult.Error != nil {
		return fmt.Errorf("保存虚拟机 XML 失败: %s", defineResult.Stderr)
	}

	return nil
}
