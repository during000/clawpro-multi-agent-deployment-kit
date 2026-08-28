package model

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"gorm.io/gorm"
)

// SMHSpace represents a space within an SMH media library.
// A single Library can have multiple Spaces, each identified by a unique SpaceTag.
// Admin/Read Token 由 smh-token-refresh 定时任务刷新后持久化于此表，
// controller 进程直接从 DB 读取，无需进程内缓存。
type SMHSpace struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	Identifier string    `gorm:"uniqueIndex:idx_space_tag_identifier;index;default:''" json:"-"` // 多租户标识，MySQL 模式下自动填充和过滤
	CreatedAt  time.Time `json:"created_at"`
	SpaceTag   string    `gorm:"uniqueIndex:idx_space_tag_identifier;not null" json:"space_tag"` // Space purpose tag, e.g. "skillhub", "common"
	SpaceId    string    `gorm:"not null" json:"space_id"`                                       // Space ID returned by SMH API
	LibraryId  string    `gorm:"not null;default:''" json:"library_id"`                          // Library ID this space belongs to
	Purpose    string    `gorm:"not null;default:''" json:"purpose"`                             // Space 用途描述，与 SpaceTag 一致

	// ── SMH Access Token（由 smh-token-refresh 定时任务维护，参考 STS 凭证持久化模式）──
	AdminToken          string `gorm:"default:''" json:"-"` // space_admin 读写 Token
	AdminTokenExpiredAt int64  `gorm:"default:0" json:"-"`  // AdminToken 过期时间（Unix 秒）
	ReadToken           string `gorm:"default:''" json:"-"` // 只读 Token
	ReadTokenExpiredAt  int64  `gorm:"default:0" json:"-"`  // ReadToken 过期时间（Unix 秒）
}

// GetSMHSpace returns the SpaceId for the given spaceTag, or empty string if not found.
func GetSMHSpace(ctx context.Context, spaceTag string) string {
	var space SMHSpace
	err := DB(ctx).Where("space_tag = ?", spaceTag).First(&space).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			slog.Error("查询 SMH Space 失败", "space_tag", spaceTag, "error", err)
		}
		return ""
	}
	return space.SpaceId
}

// GetSMHSpaceRecord returns the full SMHSpace record for the given spaceTag.
func GetSMHSpaceRecord(ctx context.Context, spaceTag string) (SMHSpace, bool) {
	var space SMHSpace
	err := DB(ctx).Where("space_tag = ?", spaceTag).First(&space).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			slog.Error("查询 SMH Space 失败", "space_tag", spaceTag, "error", err)
		}
		return space, false
	}
	return space, true
}

// UpdateSMHSpaceToken 更新指定 space 的 token（admin 或 read）。
func UpdateSMHSpaceToken(ctx context.Context, spaceTag string, admin bool, token string, expiredAt int64) error {
	fields := map[string]interface{}{}
	if admin {
		fields["admin_token"] = token
		fields["admin_token_expired_at"] = expiredAt
	} else {
		fields["read_token"] = token
		fields["read_token_expired_at"] = expiredAt
	}
	return DB(ctx).Model(&SMHSpace{}).Where("space_tag = ?", spaceTag).Updates(fields).Error
}

// UpsertSMHSpace creates or updates a Space record (idempotent by spaceTag).
func UpsertSMHSpace(ctx context.Context, spaceTag, spaceId, libraryId string) error {
	var space SMHSpace
	result := DB(ctx).Where("space_tag = ?", spaceTag).First(&space)
	if result.Error != nil {
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return result.Error
		}
		// Does not exist, create
		return DB(ctx).Create(&SMHSpace{SpaceTag: spaceTag, SpaceId: spaceId, LibraryId: libraryId, Purpose: spaceTag}).Error
	}
	// Already exists, update SpaceId and LibraryId
	return DB(ctx).Model(&space).Updates(map[string]interface{}{"space_id": spaceId, "library_id": libraryId}).Error
}

// SMHPersonalSpace represents a personal SMH space bound to a specific instance.
type SMHPersonalSpace struct {
	gorm.Model
	// 多租户标识，MySQL 模式下自动填充和过滤；与 InstanceId 组成联合唯一索引，确保不同租户的 instance_id 不冲突
	Identifier string `gorm:"uniqueIndex:idx_personal_space_instance_identifier;index;default:''" json:"-"`
	SpaceId    string `gorm:"not null" json:"space_id"`      // SMH 个人空间 ID
	UserId     uint   `gorm:"not null;index" json:"user_id"` // 所属用户 ID
	// 绑定的实例数据库 ID；与 Identifier 组成联合唯一索引（同一租户下一个实例只能绑定一个个人空间）
	InstanceId       uint       `gorm:"not null;uniqueIndex:idx_personal_space_instance_identifier" json:"instance_id"`
	UserName         string     `gorm:"not null;default:''" json:"username"`                                                             // 用户名快照（防止用户删除后丢失）
	InstanceName     string     `gorm:"not null;default:''" json:"instance_name"`                                                        // 实例名快照（防止实例删除后丢失）
	CVMInstanceId    string     `gorm:"not null;default:''" json:"cvm_instance_id"`                                                      // CVM 实例 ID
	StorageQuota     int64      `gorm:"not null;default:0" json:"storage_quota"`                                                         // 空间存储配额（单位：字节）
	FreeStorageQuota int64      `gorm:"not null;default:0" json:"free_storage_quota"`                                                    // 免费配额部分（单位：字节）
	EnvInitialized   bool       `gorm:"not null;default:false;index:idx_smh_personal_spaces_env_sync,priority:1" json:"env_initialized"` // Skill 下发且 Token 首次注入均成功
	EnvProvisionRev  int        `gorm:"not null;default:0;index:idx_smh_personal_spaces_env_sync,priority:2" json:"env_provision_rev"`   // Skill/脚本版本修订号，落后于当前目标 rev 时会重跑初始化脚本；不主动对外 admin 接口透出
	ExpiresAt        *time.Time `gorm:"default:null" json:"expires_at"`                                                                  // 个人空间过期时间（过期后收费，非删除），当前默认绑定后 3 个月
	ToBeDeletedAt    *time.Time `gorm:"default:null" json:"to_be_deleted_at"`                                                            // 计划删除时间，非空表示在回收站中
	// LastPushedTokenExpiresAt 最近一次成功下发到 CVM 的 token 的过期时间。
	// NULL 表示从未成功下发（或字段刚引入），task 层视为需要下发；
	LastPushedTokenExpiresAt *time.Time `gorm:"default:null" json:"last_pushed_token_expires_at"`
}

// CreatePersonalSpace 创建个人空间记录。ctx 用于多租户标识注入。
func CreatePersonalSpace(ctx context.Context, space *SMHPersonalSpace) error {
	return DB(ctx).Create(space).Error
}

// HasPersonalSpace 检查实例是否已绑定个人空间。ctx 用于多租户标识注入。
func HasPersonalSpace(ctx context.Context, instanceId uint) (bool, error) {
	var count int64
	err := DB(ctx).Model(&SMHPersonalSpace{}).Where("instance_id = ?", instanceId).Count(&count).Error
	return count > 0, err
}
