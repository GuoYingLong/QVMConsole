package service

import (
	"context"

	"kvm_console/model"
	publicippkg "kvm_console/service/public_ip"
)

// PublicIP function delegates - forward to service/public_ip subpackage
// Maintains backward compatibility for callers using service.XXX()

func ListPublicIPs() ([]PublicIPInfo, error) {
	return publicippkg.ListPublicIPs()
}

func CreatePublicIP(req PublicIPRequest) (*model.PublicIP, error) {
	return publicippkg.CreatePublicIP(req)
}

func UpdatePublicIP(id uint, req PublicIPRequest) (*model.PublicIP, error) {
	return publicippkg.UpdatePublicIP(id, req)
}

func DeletePublicIP(id uint) error {
	return publicippkg.DeletePublicIP(id)
}

func PreviewPublicIPBinding(id uint, req PublicIPBindRequest) (*PublicIPPreview, error) {
	return publicippkg.PreviewPublicIPBinding(id, req)
}

func ExecutePublicIPOperation(ctx context.Context, params PublicIPOperationParams, progress func(int, string)) (string, error) {
	return publicippkg.ExecutePublicIPOperation(ctx, params, progress)
}

func ApplyPublicIPRules() error {
	return publicippkg.ApplyPublicIPRules()
}

func RestorePublicIPRules() error {
	return publicippkg.RestorePublicIPRules()
}

func BuildPublicIPRulesScript() (string, error) {
	return publicippkg.BuildPublicIPRulesScript()
}

func ResolvePublicIPVMPrivateIP(vmName string) string {
	return publicippkg.ResolvePublicIPVMPrivateIP(vmName)
}

func PublicIPNATPrivateIPsForVM(vmName string) []string {
	return publicippkg.PublicIPNATPrivateIPsForVM(vmName)
}

func GetUserPublicIPUsage(username string) int {
	return publicippkg.GetUserPublicIPUsage(username)
}

func NormalizePublicIPMode(mode string) string {
	return publicippkg.NormalizePublicIPMode(mode)
}

func PublicIPModeLabel(mode string) string {
	return publicippkg.PublicIPModeLabel(mode)
}

func ListPublicIPAttachmentsForVM(vmName string) []PublicIPAttachment {
	return publicippkg.ListPublicIPAttachmentsForVM(vmName)
}

func ParsePublicIPOperationParams(raw string) (PublicIPOperationParams, error) {
	return publicippkg.ParsePublicIPOperationParams(raw)
}

func ParsePublicIPID(raw string) (uint, error) {
	return publicippkg.ParsePublicIPID(raw)
}
