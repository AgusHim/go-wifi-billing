package dto

type CreateCustomerDTO struct {
	Name          *string  `json:"name,omitempty" validate:"omitempty,max=255"`
	Email         *string  `json:"email,omitempty" validate:"omitempty,email"`
	Phone         *string  `json:"phone,omitempty" validate:"omitempty,e164|numeric,min=8,max=15"`
	Password      *string  `json:"password,omitempty" validate:"omitempty,min=6"`
	CoverageID    *string  `json:"coverage_id,omitempty" validate:"omitempty,uuid4"`
	OdcID         *string  `json:"odc_id,omitempty" validate:"omitempty,uuid4"`
	OdpID         *string  `json:"odp_id,omitempty" validate:"omitempty,uuid4"`
	PortOdp       *string  `json:"port_odp,omitempty"`
	ServiceNumber *string  `json:"service_number,omitempty"`
	Card          *string  `json:"card,omitempty"`
	IDCard        *string  `json:"id_card,omitempty" validate:"omitempty,len=16,numeric"`
	IsSendWA      *bool    `json:"is_send_wa,omitempty"`
	Status        *string  `json:"status,omitempty" validate:"omitempty"`
	Address       *string  `json:"address,omitempty" validate:"omitempty,max=255"`
	Description   *string  `json:"description,omitempty" validate:"omitempty,max=255"`
	Latitude      *float64 `json:"latitude,omitempty" validate:"omitempty,min=-90,max=90"`
	Longitude     *float64 `json:"longitude,omitempty" validate:"omitempty,min=-180,max=180"`
	Mode          *string  `json:"mode,omitempty" validate:"omitempty"`
	IDPPOE        *string  `json:"id_ppoe,omitempty"`
	ProfilePPOE   *string  `json:"profile_ppoe,omitempty"`
	AdminID       *string  `json:"admin_id,omitempty" validate:"omitempty,uuid4"`
}
