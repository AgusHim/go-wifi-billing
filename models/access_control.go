package models

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	PermissionEffectAllow = "allow"
	PermissionEffectDeny  = "deny"
)

// CanonicalRoleKey normalizes legacy role aliases while users.role is still
// retained during the RBAC migration.
func CanonicalRoleKey(role string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "root", "owner":
		return "owner", true
	case "admin":
		return "admin", true
	case "petugas":
		return "petugas", true
	case "loket":
		return "loket", true
	case "technician", "teknisi", "noc":
		return "teknisi", true
	case "user", "customer":
		return "customer", true
	default:
		return "", false
	}
}

// Role is the canonical access profile assigned to a user. User.Role remains
// available temporarily as a legacy compatibility field during RBAC rollout.
type Role struct {
	ID          uuid.UUID  `json:"id" gorm:"type:uuid;primaryKey"`
	Key         string     `json:"key" gorm:"type:varchar(50);not null;unique"`
	Name        string     `json:"name" gorm:"type:varchar(100);not null"`
	Description string     `json:"description" gorm:"type:text"`
	IsSystem    bool       `json:"is_system" gorm:"not null;default:false;check:chk_roles_owner_is_system,NOT is_owner OR is_system"`
	IsOwner     bool       `json:"is_owner" gorm:"not null;default:false;uniqueIndex:idx_roles_single_owner,where:is_owner = true"`
	IsActive    bool       `json:"is_active" gorm:"not null;default:true;index"`
	CreatedBy   *uuid.UUID `json:"created_by,omitempty" gorm:"type:uuid;index"`
	UpdatedBy   *uuid.UUID `json:"updated_by,omitempty" gorm:"type:uuid;index"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func (r *Role) BeforeCreate(tx *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}

type Permission struct {
	ID          uuid.UUID `json:"id" gorm:"type:uuid;primaryKey"`
	Key         string    `json:"key" gorm:"type:varchar(100);not null;unique"`
	Module      string    `json:"module" gorm:"type:varchar(50);not null;index:idx_permissions_module_sort,priority:1"`
	Action      string    `json:"action" gorm:"type:varchar(50);not null"`
	Name        string    `json:"name" gorm:"type:varchar(150);not null"`
	Description string    `json:"description" gorm:"type:text"`
	RiskLevel   string    `json:"risk_level" gorm:"type:varchar(20);not null;default:low;check:chk_permissions_risk_level,risk_level IN ('low','medium','high','critical')"`
	SortOrder   int       `json:"sort_order" gorm:"not null;default:0;index:idx_permissions_module_sort,priority:2"`
	IsSystem    bool      `json:"is_system" gorm:"not null;default:true"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (p *Permission) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}

type RolePermission struct {
	RoleID       uuid.UUID  `json:"role_id" gorm:"type:uuid;primaryKey;index:idx_role_permissions_permission_role,priority:2"`
	PermissionID uuid.UUID  `json:"permission_id" gorm:"type:uuid;primaryKey;index:idx_role_permissions_permission_role,priority:1"`
	CreatedBy    *uuid.UUID `json:"created_by,omitempty" gorm:"type:uuid;index"`
	CreatedAt    time.Time  `json:"created_at"`

	Role       *Role       `json:"role,omitempty" gorm:"foreignKey:RoleID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Permission *Permission `json:"permission,omitempty" gorm:"foreignKey:PermissionID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

type UserPermissionOverride struct {
	UserID       uuid.UUID `json:"user_id" gorm:"type:uuid;primaryKey;index:idx_user_permission_overrides_permission_user,priority:2"`
	PermissionID uuid.UUID `json:"permission_id" gorm:"type:uuid;primaryKey;index:idx_user_permission_overrides_permission_user,priority:1"`
	Effect       string    `json:"effect" gorm:"type:varchar(10);not null;check:chk_user_permission_overrides_effect,effect IN ('allow','deny')"`
	Reason       string    `json:"reason" gorm:"type:text"`
	CreatedBy    uuid.UUID `json:"created_by" gorm:"type:uuid;not null;index"`
	UpdatedBy    uuid.UUID `json:"updated_by" gorm:"type:uuid;not null;index"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`

	User       *User       `json:"user,omitempty" gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Permission *Permission `json:"permission,omitempty" gorm:"foreignKey:PermissionID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

type AccessAuditLog struct {
	ID          uuid.UUID       `json:"id" gorm:"type:uuid;primaryKey"`
	ActorUserID uuid.UUID       `json:"actor_user_id" gorm:"type:uuid;not null;index:idx_access_audit_actor_created,priority:1"`
	TargetType  string          `json:"target_type" gorm:"type:varchar(20);not null;index:idx_access_audit_target_created,priority:1;check:chk_access_audit_target_type,target_type IN ('role','user')"`
	TargetID    uuid.UUID       `json:"target_id" gorm:"type:uuid;not null;index:idx_access_audit_target_created,priority:2"`
	Action      string          `json:"action" gorm:"type:varchar(80);not null;index"`
	BeforeData  json.RawMessage `json:"before_data,omitempty" gorm:"type:jsonb"`
	AfterData   json.RawMessage `json:"after_data,omitempty" gorm:"type:jsonb"`
	Reason      string          `json:"reason" gorm:"type:text"`
	IPAddress   string          `json:"ip_address" gorm:"type:varchar(64)"`
	UserAgent   string          `json:"user_agent" gorm:"type:text"`
	RequestID   string          `json:"request_id" gorm:"type:varchar(100);index"`
	CreatedAt   time.Time       `json:"created_at" gorm:"index:idx_access_audit_actor_created,priority:2;index:idx_access_audit_target_created,priority:3"`

	Actor *User `json:"actor,omitempty" gorm:"foreignKey:ActorUserID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
}

func (a *AccessAuditLog) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}
