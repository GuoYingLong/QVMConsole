package service

import (
	userpkg "kvm_console/service/user"
)

// ── Type aliases for structs moved to service/user ──

type VMUserInfo = userpkg.VMUserInfo
type UserStatusChangeResult = userpkg.UserStatusChangeResult
type QuotaUsage = userpkg.QuotaUsage
type UserStorageInfo = userpkg.UserStorageInfo
type UserFileInfo = userpkg.UserFileInfo
type UserRuntimeQuotaSnapshot = userpkg.UserRuntimeQuotaSnapshot
type RuntimeQuotaShutdownResult = userpkg.RuntimeQuotaShutdownResult

// ── Constant aliases ──

const (
	StorageCategoryISO   = userpkg.StorageCategoryISO
	StorageCategoryShare = userpkg.StorageCategoryShare
	StorageCategoryDisk  = userpkg.StorageCategoryDisk
)
