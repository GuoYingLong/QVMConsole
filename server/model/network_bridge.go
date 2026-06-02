package model

import "time"

// NetworkBridge 记录面板管理的宿主机网桥。
type NetworkBridge struct {
	ID            uint      `json:"id" gorm:"primaryKey"`
	Name          string    `json:"name" gorm:"uniqueIndex;not null;size:64"`
	Mode          string    `json:"mode" gorm:"not null;size:16"` // nat/bridge
	UplinkIF      string    `json:"uplink_if" gorm:"size:64"`
	MigrateHostIP bool      `json:"migrate_host_ip" gorm:"default:false"`
	IsDefault     bool      `json:"is_default" gorm:"default:false"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (NetworkBridge) TableName() string {
	return "network_bridges"
}
