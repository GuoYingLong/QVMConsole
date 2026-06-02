package service

import (
	crand "crypto/rand"
	"fmt"
	"math/big"
	"regexp"
	"strings"
	"time"
)

const strongPasswordMinLength = 12
const windowsCloneDefaultUsername = "administrator"

var (
	cloneHostnameRegexp       = regexp.MustCompile(`^[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?$`)
	cloneUsernameRegexp       = regexp.MustCompile(`^[a-z_][a-z0-9_-]{0,31}$`)
	clonePasswordAllowedRegex = regexp.MustCompile(`^[A-Za-z0-9!@#$%^&*_\-+=?]+$`)
	clonePasswordUpperRegex   = regexp.MustCompile(`[A-Z]`)
	clonePasswordLowerRegex   = regexp.MustCompile(`[a-z]`)
	clonePasswordDigitRegex   = regexp.MustCompile(`[0-9]`)
	clonePasswordSymbolRegex  = regexp.MustCompile(`[!@#$%^&*_\-+=?]`)
)

// ValidateCloneCredentials 校验模板克隆使用的主机名、用户名和密码
func ValidateCloneCredentials(hostname, username, password string, requireCredentials bool) error {
	trimmedHostname := strings.TrimSpace(hostname)
	if trimmedHostname != "" && !cloneHostnameRegexp.MatchString(trimmedHostname) {
		return fmt.Errorf("主机名只能包含字母、数字和短横线，且不能以短横线开头或结尾")
	}

	trimmedUsername := strings.TrimSpace(username)
	if requireCredentials && trimmedUsername == "" {
		return fmt.Errorf("请输入用户名")
	}
	if trimmedUsername != "" && !cloneUsernameRegexp.MatchString(trimmedUsername) {
		return fmt.Errorf("用户名只能以小写字母或下划线开头，且只能包含小写字母、数字、下划线和短横线")
	}

	if requireCredentials && password == "" {
		return fmt.Errorf("请输入密码")
	}
	if password != "" {
		return ValidateStrongPassword(password)
	}

	return nil
}

// NormalizeCloneUsernameForTemplate 根据模板类型补全默认用户名
func NormalizeCloneUsernameForTemplate(templateType, username string) string {
	trimmedTemplateType := strings.ToLower(strings.TrimSpace(templateType))
	trimmedUsername := strings.TrimSpace(username)
	if trimmedTemplateType == "windows" && trimmedUsername == "" {
		return windowsCloneDefaultUsername
	}
	return trimmedUsername
}

// ValidateCloneCredentialsForTemplate 校验模板克隆使用的主机名、用户名和密码
func ValidateCloneCredentialsForTemplate(templateType, hostname, username, password string, requireCredentials bool) error {
	trimmedTemplateType := strings.ToLower(strings.TrimSpace(templateType))
	normalizedUsername := NormalizeCloneUsernameForTemplate(trimmedTemplateType, username)
	if trimmedTemplateType == "windows" && normalizedUsername != windowsCloneDefaultUsername {
		return fmt.Errorf("Windows 模板用户名固定为 administrator，不支持修改")
	}
	return ValidateCloneCredentials(hostname, normalizedUsername, password, requireCredentials)
}

// ValidateStrongPassword 校验模板克隆使用的强密码
func ValidateStrongPassword(password string) error {
	if len(password) < strongPasswordMinLength {
		return fmt.Errorf("密码长度不能少于%d位", strongPasswordMinLength)
	}
	if !clonePasswordAllowedRegex.MatchString(password) {
		return fmt.Errorf("密码只能包含字母、数字和 !@#$%%^&*_-+=? 符号")
	}
	if !clonePasswordUpperRegex.MatchString(password) ||
		!clonePasswordLowerRegex.MatchString(password) ||
		!clonePasswordDigitRegex.MatchString(password) ||
		!clonePasswordSymbolRegex.MatchString(password) {
		return fmt.Errorf("密码必须同时包含大写字母、小写字母、数字和符号")
	}
	return nil
}

// GenerateRandomCloneHostname 生成默认主机名
func GenerateRandomCloneHostname() string {
	return fmt.Sprintf("vm-%s", randomStringFromCharset("abcdefghijklmnopqrstuvwxyz0123456789", 8))
}

// randomStringFromCharset 生成指定字符集的随机字符串
func randomStringFromCharset(charset string, length int) string {
	if length <= 0 || charset == "" {
		return ""
	}

	var builder strings.Builder
	builder.Grow(length)

	max := big.NewInt(int64(len(charset)))
	for index := 0; index < length; index++ {
		randomIndex, err := crand.Int(crand.Reader, max)
		if err != nil {
			fallbackIndex := int((time.Now().UnixNano() + int64(index)) % int64(len(charset)))
			if fallbackIndex < 0 {
				fallbackIndex = -fallbackIndex
			}
			builder.WriteByte(charset[fallbackIndex])
			continue
		}
		builder.WriteByte(charset[randomIndex.Int64()])
	}

	return builder.String()
}

// GenerateRandomStrongPassword 生成随机强密码（至少12位，含大小写字母、数字和符号）
func GenerateRandomStrongPassword() string {
	upper := "ABCDEFGHIJKLMNPQRSTUVWXYZ"
	lower := "abcdefghijkmnpqrstuvwxyz"
	digits := "23456789"
	symbols := "!@#$%^&*_+=?"

	charsets := []string{upper, lower, digits, symbols}
	var parts []string
	for _, cs := range charsets {
		parts = append(parts, randomStringFromCharset(cs, 3))
	}

	return strings.Join(parts, "")
}
