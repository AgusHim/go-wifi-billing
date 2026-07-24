package dto

type RegisterDTO struct {
	Name       string  `json:"name" validate:"required"`
	Email      string  `json:"email" validate:"required,email"`
	Phone      string  `json:"phone"`
	Password   string  `json:"password" validate:"required,min=6"`
	CoverageID *string `json:"coverage_id"`
}

type LoginDTO struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type CreateUserDTO struct {
	Name     string `json:"name" validate:"required"`
	Email    string `json:"email" validate:"required,email"`
	Phone    string `json:"phone"`
	Password string `json:"password" validate:"required,min=6"`
	Role     string `json:"role" validate:"required"`
}

type UpdateProfileDTO struct {
	Name  *string `json:"name"`
	Email *string `json:"email"`
	Phone *string `json:"phone"`
}

type UpdateUserDTO struct {
	Name     *string `json:"name"`
	Email    *string `json:"email"`
	Phone    *string `json:"phone"`
	Password *string `json:"password"`
}

// UpdateUserAccessDTO is intentionally separate from profile/user payloads.
// The owner-only access-control API introduced in Phase 3 will consume it.
type UpdateUserAccessDTO struct {
	RoleID                    string                      `json:"role_id"`
	Overrides                 []UserPermissionOverrideDTO `json:"overrides"`
	Reason                    string                      `json:"reason"`
	ExpectedPermissionVersion int64                       `json:"expected_permission_version"`
}

type UserPermissionOverrideDTO struct {
	PermissionKey string `json:"permission_key"`
	Effect        string `json:"effect"`
}
