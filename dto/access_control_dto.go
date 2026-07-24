package dto

type CreateRoleDTO struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type UpdateRoleDTO struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	IsActive    *bool   `json:"is_active"`
	Reason      string  `json:"reason"`
}

type UpdateRolePermissionsDTO struct {
	PermissionKeys []string `json:"permission_keys"`
	Reason         string   `json:"reason"`
}

type ResetUserOverridesDTO struct {
	ExpectedPermissionVersion int64  `json:"expected_permission_version"`
	Reason                    string `json:"reason"`
}

type AccessChangeMetadata struct {
	IPAddress string
	UserAgent string
	RequestID string
}
