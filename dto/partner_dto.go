package dto

import "time"

type PartnerRequest struct {
	Code, Name, BillingEmail, Phone, ContactName, Address, TaxNumber string
	IsActive                                                         *bool `json:"is_active"`
}
type PartnerMemberRequest struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
}
type PartnerBandwidthServiceRequest struct {
	Name, Capacity, BillingMode, Status string
	Rate, DueDay                        int
	StartsAt, EndsAt                    *time.Time `json:"starts_at"`
}
type PartnerInvoiceLineRequest struct {
	Description, LineType, Unit string
	Quantity                    float64 `json:"quantity"`
	UnitPrice, TaxAmount        int     `json:"unit_price"`
}
type PartnerInvoiceRequest struct {
	PartnerID string                      `json:"partner_id"`
	DueDate   time.Time                   `json:"due_date"`
	Lines     []PartnerInvoiceLineRequest `json:"lines"`
}
type PartnerPaymentRequest struct {
	Amount                      int `json:"amount"`
	Method, Reference, ProofURL string
}
type PartnerPaymentVerificationRequest struct {
	Approve         bool   `json:"approve"`
	RejectionReason string `json:"rejection_reason"`
}
