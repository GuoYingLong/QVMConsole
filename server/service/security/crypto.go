package security

import (
	"crypto/aes"
	"crypto/cipher"
	crand "crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"

	"kvm_console/config"
)

// EncryptSecurityText 使用账户安全密钥加密敏感文本
func EncryptSecurityText(plainText string) (string, error) {
	block, err := aes.NewCipher(buildSecurityKey())
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(crand.Reader, nonce); err != nil {
		return "", err
	}

	cipherText := gcm.Seal(nil, nonce, []byte(plainText), nil)
	payload := append(nonce, cipherText...)
	return base64.StdEncoding.EncodeToString(payload), nil
}

// DecryptSecurityText 解密敏感文本
func DecryptSecurityText(cipherText string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(cipherText)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(buildSecurityKey())
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(raw) < nonceSize {
		return "", fmt.Errorf("密文格式无效")
	}

	nonce := raw[:nonceSize]
	payload := raw[nonceSize:]
	plain, err := gcm.Open(nil, nonce, payload, nil)
	if err != nil {
		return "", err
	}

	return string(plain), nil
}

func buildSecurityKey() []byte {
	sum := sha256.Sum256([]byte(config.GlobalConfig.SecuritySecret))
	return sum[:]
}
