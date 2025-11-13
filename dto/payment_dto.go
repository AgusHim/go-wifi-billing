package dto

type CreateMidtransPayment struct {
	BillID string `json:"bill_id" validate:"required"`
}
