package model

import (
	"log"
	"os"
	"path/filepath"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"kvm_console/config"
)

// DB 全局数据库实例
var DB *gorm.DB

// InitDB 初始化数据库
func InitDB() {
	// 确保数据目录存在
	dbDir := filepath.Dir(config.GlobalConfig.DBPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		log.Fatalf("创建数据库目录失败: %v", err)
	}

	var err error
	DB, err = gorm.Open(sqlite.Open(config.GlobalConfig.DBPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}

	hadMaxPortForwardsColumn := DB.Migrator().HasColumn(&User{}, "max_port_forwards")
	hadEnablePortForwardColumn := DB.Migrator().HasColumn(&User{}, "enable_port_forward")
	hadUserMaxSnapshotsColumn := DB.Migrator().HasColumn(&User{}, "max_snapshots")
	hadLightweightQuotaMaxSnapshotsColumn := DB.Migrator().HasColumn(&LightweightVMQuota{}, "max_snapshots")
	hadLightweightRegistrationMaxSnapshotsColumn := DB.Migrator().HasColumn(&LightweightVMRegistration{}, "max_snapshots")
	hadLightweightQuotaMaxRuntimeColumn := DB.Migrator().HasColumn(&LightweightVMQuota{}, "max_runtime_hours")
	hadLightweightRegistrationMaxRuntimeColumn := DB.Migrator().HasColumn(&LightweightVMRegistration{}, "max_runtime_hours")

	// 自动迁移表结构
	if err := DB.AutoMigrate(&User{}, &UserAPIKey{}, &VmStatsRecord{}, &PortForwardIP{}, &PortForwardWhitelist{}, &PortForwardProbeState{}, &HostStatsRecord{}, &UserTrafficDaily{}, &SystemSetting{}, &VMCredential{}, &VMCache{}, &AuthActionToken{}, &SecurityChallenge{}, &SchedulerEvent{}, &VMSchedule{}, &NetworkBridge{}, &HostStoragePool{}, &HostNode{},
		&LightweightVMQuota{}, &LightweightVMTrafficMonthly{}, &LightweightVMRegistration{},
		&VPCSwitch{}, &VPCSecurityGroup{}, &VPCSecurityGroupRule{}, &VPCVMBinding{}, &VPCSwitchTrafficMonthly{}, &PublicIP{}, &PublicIPBinding{},
		&VMLock{}); err != nil {
		log.Fatalf("数据库迁移失败: %v", err)
	}
	migrateUserCloudType()
	migratePublicIPCIDRColumn()
	migrateUserPortForwardFeature(hadEnablePortForwardColumn)
	migrateUserPortForwardQuota(hadMaxPortForwardsColumn)
	migrateUserSnapshotQuota(hadUserMaxSnapshotsColumn)
	migrateLightweightSnapshotQuota(hadLightweightQuotaMaxSnapshotsColumn, hadLightweightRegistrationMaxSnapshotsColumn)
	migrateLightweightRuntimeQuota(hadLightweightQuotaMaxRuntimeColumn, hadLightweightRegistrationMaxRuntimeColumn)

	// 兼容旧用户：补齐默认状态，确保升级后能继续登录
	if err := DB.Model(&User{}).Where("status = '' OR status IS NULL").Updates(map[string]interface{}{
		"status": "active",
	}).Error; err != nil {
		log.Printf("修复旧用户状态失败: %v", err)
	}

	// 初始化默认管理员
	initDefaultAdmin()
	log.Println("数据库初始化完成")
}

func migrateUserCloudType() {
	if DB == nil {
		return
	}
	if err := DB.Model(&User{}).
		Where("cloud_type = '' OR cloud_type IS NULL").
		Update("cloud_type", "elastic").Error; err != nil {
		log.Printf("初始化用户云类型失败: %v", err)
	}
}

func migrateUserPortForwardQuota(hadColumn bool) {
	if DB == nil || hadColumn {
		return
	}
	if err := DB.Model(&User{}).
		Where("role <> ? AND (max_port_forwards IS NULL OR max_port_forwards = 0)", "admin").
		Update("max_port_forwards", 10).Error; err != nil {
		log.Printf("初始化用户端口转发配额失败: %v", err)
	}
}

func migrateUserPortForwardFeature(hadColumn bool) {
	if DB == nil || hadColumn {
		return
	}
	if err := DB.Model(&User{}).
		Where("role <> ?", "admin").
		Update("enable_port_forward", true).Error; err != nil {
		log.Printf("初始化用户端口转发开关失败: %v", err)
	}
}

func migrateUserSnapshotQuota(hadColumn bool) {
	if DB == nil || hadColumn {
		return
	}
	if err := DB.Model(&User{}).
		Where("role <> ? AND (max_snapshots IS NULL OR max_snapshots = 0)", "admin").
		Update("max_snapshots", 5).Error; err != nil {
		log.Printf("初始化用户快照配额失败: %v", err)
	}
}

func migrateLightweightSnapshotQuota(hadQuotaColumn, hadRegistrationColumn bool) {
	if DB == nil {
		return
	}
	if !hadQuotaColumn {
		if err := DB.Model(&LightweightVMQuota{}).
			Where("max_snapshots IS NULL OR max_snapshots = 0").
			Update("max_snapshots", 2).Error; err != nil {
			log.Printf("初始化轻量云 VM 快照配额失败: %v", err)
		}
	}
	if !hadRegistrationColumn {
		if err := DB.Model(&LightweightVMRegistration{}).
			Where("max_snapshots IS NULL OR max_snapshots = 0").
			Update("max_snapshots", 2).Error; err != nil {
			log.Printf("初始化轻量云 VM 注册快照配额失败: %v", err)
		}
	}
}

func migrateLightweightRuntimeQuota(hadQuotaColumn, hadRegistrationColumn bool) {
	if DB == nil {
		return
	}
	if !hadQuotaColumn {
		if err := DB.Model(&LightweightVMQuota{}).
			Where("max_runtime_hours IS NULL").
			Update("max_runtime_hours", 0).Error; err != nil {
			log.Printf("初始化轻量云 VM 运行时长配额失败: %v", err)
		}
	}
	if !hadRegistrationColumn {
		if err := DB.Model(&LightweightVMRegistration{}).
			Where("max_runtime_hours IS NULL").
			Update("max_runtime_hours", 0).Error; err != nil {
			log.Printf("初始化轻量云 VM 注册运行时长配额失败: %v", err)
		}
	}
}

func migratePublicIPCIDRColumn() {
	if DB == nil || !DB.Migrator().HasTable(&PublicIP{}) {
		return
	}
	if !DB.Migrator().HasColumn(&PublicIP{}, "c_id_r") || !DB.Migrator().HasColumn(&PublicIP{}, "cidr") {
		return
	}
	if err := DB.Exec("UPDATE public_ips SET cidr = c_id_r WHERE (cidr IS NULL OR cidr = '') AND c_id_r IS NOT NULL AND c_id_r <> ''").Error; err != nil {
		log.Printf("迁移公网 IP CIDR 字段失败: %v", err)
	}
}

// initDefaultAdmin 创建默认管理员账号
func initDefaultAdmin() {
	var count int64
	DB.Model(&User{}).Where("role = ?", "admin").Count(&count)
	if count > 0 {
		return
	}

	// 密码加密
	hashedPassword, err := bcrypt.GenerateFromPassword(
		[]byte(config.GlobalConfig.DefaultAdminPass), bcrypt.DefaultCost,
	)
	if err != nil {
		log.Fatalf("生成密码哈希失败: %v", err)
	}

	admin := User{
		Username:     config.GlobalConfig.DefaultAdminUser,
		PasswordHash: string(hashedPassword),
		Role:         "admin",
		Status:       "active",
	}

	if err := DB.Create(&admin).Error; err != nil {
		log.Printf("创建默认管理员失败: %v", err)
	} else {
		log.Printf("默认管理员账号已创建: %s", admin.Username)
	}
}
