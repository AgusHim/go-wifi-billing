package dto

import "time"

type PartnerRequest struct {
	Code         string `json:"code"`
	Name         string `json:"name"`
	BillingEmail string `json:"billing_email"`
	Phone        string `json:"phone"`
	ContactName  string `json:"contact_name"`
	Address      string `json:"address"`
	TaxNumber    string `json:"tax_number"`
	IsActive     *bool  `json:"is_active"`
}
type PartnerMemberRequest struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
}
type PartnerBandwidthServiceRequest struct {
	Name        string     `json:"name"`
	Capacity    string     `json:"capacity"`
	BillingMode string     `json:"billing_mode"`
	Status      string     `json:"status"`
	Rate        int        `json:"rate"`
	DueDay      int        `json:"due_day"`
	StartsAt    *time.Time `json:"starts_at"`
	EndsAt      *time.Time `json:"ends_at"`
}
type PartnerInvoiceLineRequest struct {
	Description string  `json:"description"`
	LineType    string  `json:"line_type"`
	Unit        string  `json:"unit"`
	Quantity    float64 `json:"quantity"`
	UnitPrice   int     `json:"unit_price"`
	TaxAmount   int     `json:"tax_amount"`
	// PeriodYear/PeriodMonth tag a bandwidth line with the service period so the
	// same period cannot be billed twice.
	PeriodYear  *int   `json:"period_year"`
	PeriodMonth *int   `json:"period_month"`
	ServiceID   string `json:"service_id"`
	// AllocationID links an inventory line to the allocation it bills.
	AllocationID string `json:"allocation_id"`
}

type PartnerInvoiceRequest struct {
	PartnerID string                      `json:"partner_id"`
	DueDate   time.Time                   `json:"due_date"`
	Lines     []PartnerInvoiceLineRequest `json:"lines"`
}

// PartnerInvoiceFromServiceRequest generates a draft invoice for one service
// period. `amount` is only used for the manual billing mode.
type PartnerInvoiceFromServiceRequest struct {
	PartnerID   string     `json:"partner_id"`
	ServiceID   string     `json:"service_id"`
	PeriodYear  int        `json:"period_year"`
	PeriodMonth int        `json:"period_month"`
	Amount      int        `json:"amount"`
	TaxAmount   int        `json:"tax_amount"`
	DueDate     *time.Time `json:"due_date"`
	Notes       string     `json:"notes"`
}

type PartnerBandwidthUsageRequest struct {
	ServiceID   string  `json:"service_id"`
	PeriodYear  int     `json:"period_year"`
	PeriodMonth int     `json:"period_month"`
	Quantity    float64 `json:"quantity"`
	Unit        string  `json:"unit"`
	DataSource  string  `json:"data_source"`
	EvidenceURL string  `json:"evidence_url"`
	Notes       string  `json:"notes"`
}

type PartnerBandwidthUsageApprovalRequest struct {
	Approve      bool   `json:"approve"`
	RejectReason string `json:"reject_reason"`
}

type PartnerAllocationRequest struct {
	ItemID          string  `json:"item_id"`
	SerialItemID    string  `json:"serial_item_id"`
	Quantity        float64 `json:"quantity"`
	UnitPrice       int     `json:"unit_price"`
	FromLocationID  string  `json:"from_location_id"`
	TransactionType string  `json:"transaction_type"` // loan, sale
	Notes           string  `json:"notes"`
}

type PartnerAllocationReturnRequest struct {
	ToLocationID string `json:"to_location_id"`
	Notes        string `json:"notes"`
}

type LegalDocumentRequest struct {
	Name             string     `json:"name"`
	Category         string     `json:"category"`
	Version          string     `json:"version"`
	EffectiveDate    *time.Time `json:"effective_date"`
	ExpiresAt        *time.Time `json:"expires_at"`
	FilePath         string     `json:"file_path"`
	MimeType         string     `json:"mime_type"`
	FileSize         int64      `json:"file_size"`
	Status           string     `json:"status"`
	IsPartnerVisible bool       `json:"is_partner_visible"`
	Notes            string     `json:"notes"`
}

type PartnerPaymentRequest struct {
	Amount         int    `json:"amount"`
	Method         string `json:"method"`
	Reference      string `json:"reference"`
	ProofURL       string `json:"proof_url"`
	IdempotencyKey string `json:"idempotency_key"`
}

type PartnerPaymentVerificationRequest struct {
	Approve         bool   `json:"approve"`
	RejectionReason string `json:"rejection_reason"`
}

// BrevoWebhookEvent is the subset of a Brevo transactional event we persist to
// correlate delivery/bounce/complaint with a sent notification.
type BrevoWebhookEvent struct {
	Event     string `json:"event"`
	MessageID string `json:"message-id"`
	Email     string `json:"email"`
	Reason    string `json:"reason"`
	TS        int64  `json:"ts"`
}
