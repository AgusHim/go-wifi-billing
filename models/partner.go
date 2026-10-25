package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Partner is a B2B customer. Its invoices are receivables for Cantika.net.
type Partner struct {
	ID           uuid.UUID       `json:"id" gorm:"type:uuid;primaryKey"`
	Code         string          `json:"code" gorm:"type:varchar(50);not null;uniqueIndex"`
	Name         string          `json:"name" gorm:"type:varchar(255);not null"`
	BillingEmail string          `json:"billing_email" gorm:"type:varchar(255);not null"`
	Phone        string          `json:"phone" gorm:"type:varchar(50)"`
	ContactName  string          `json:"contact_name" gorm:"type:varchar(255)"`
	Address      string          `json:"address" gorm:"type:text"`
	TaxNumber    string          `json:"tax_number" gorm:"type:varchar(100)"`
	IsActive     bool            `json:"is_active" gorm:"not null;default:true"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
	DeletedAt    gorm.DeletedAt  `json:"deleted_at" gorm:"index"`
	Members      []PartnerMember `json:"members,omitempty"`
}

func (p *Partner) BeforeCreate(_ *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}

type PartnerMember struct {
	ID        uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	PartnerID uuid.UUID      `json:"partner_id" gorm:"type:uuid;not null;uniqueIndex:idx_partner_member"`
	UserID    uuid.UUID      `json:"user_id" gorm:"type:uuid;not null;uniqueIndex:idx_partner_member;index"`
	Role      string         `json:"role" gorm:"type:varchar(30);not null;default:member"`
	Status    string         `json:"status" gorm:"type:varchar(30);not null;default:active"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index"`
	Partner   Partner        `json:"partner,omitempty" gorm:"foreignKey:PartnerID"`
	User      User           `json:"user,omitempty" gorm:"foreignKey:UserID"`
}

func (p *PartnerMember) BeforeCreate(_ *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}

type PartnerBandwidthService struct {
	ID          uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	PartnerID   uuid.UUID      `json:"partner_id" gorm:"type:uuid;not null;index"`
	Name        string         `json:"name" gorm:"type:varchar(255);not null"`
	Capacity    string         `json:"capacity" gorm:"type:varchar(100)"`
	BillingMode string         `json:"billing_mode" gorm:"type:varchar(20);not null"` // fixed, usage, manual
	Rate        int            `json:"rate" gorm:"not null;default:0"`
	DueDay      int            `json:"due_day"`
	StartsAt    *time.Time     `json:"starts_at"`
	EndsAt      *time.Time     `json:"ends_at"`
	Status      string         `json:"status" gorm:"type:varchar(30);not null;default:active"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"deleted_at" gorm:"index"`
}

func (p *PartnerBandwidthService) BeforeCreate(_ *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}

type PartnerInvoice struct {
	ID                   uuid.UUID                     `json:"id" gorm:"type:uuid;primaryKey"`
	Number               string                        `json:"number" gorm:"type:varchar(80);not null;uniqueIndex"`
	PartnerID            uuid.UUID                     `json:"partner_id" gorm:"type:uuid;not null;index"`
	IssuedAt             *time.Time                    `json:"issued_at"`
	DueDate              time.Time                     `json:"due_date"`
	Status               string                        `json:"status" gorm:"type:varchar(30);not null;default:draft;index"`
	Subtotal             int                           `json:"subtotal"`
	TaxAmount            int                           `json:"tax_amount"`
	Total                int                           `json:"total"`
	VerifiedPaidAmount   int                           `json:"verified_paid_amount"`
	BalanceDue           int                           `json:"balance_due"`
	IssuerName           string                        `json:"issuer_name" gorm:"type:varchar(255)"`
	IssuerTaxNumber      string                        `json:"issuer_tax_number" gorm:"type:varchar(100)"`
	PartnerNameSnapshot  string                        `json:"partner_name_snapshot" gorm:"type:varchar(255)"`
	PartnerEmailSnapshot string                        `json:"partner_email_snapshot" gorm:"type:varchar(255)"`
	CreatedBy            *uuid.UUID                    `json:"created_by" gorm:"type:uuid"`
	CreatedAt            time.Time                     `json:"created_at"`
	UpdatedAt            time.Time                     `json:"updated_at"`
	DeletedAt            gorm.DeletedAt                `json:"deleted_at" gorm:"index"`
	Partner              Partner                       `json:"partner,omitempty" gorm:"foreignKey:PartnerID"`
	Lines                []PartnerInvoiceLine          `json:"lines,omitempty" gorm:"foreignKey:InvoiceID"`
	Payments             []PartnerPayment              `json:"payments,omitempty" gorm:"foreignKey:InvoiceID"`
	StatusHistory        []PartnerInvoiceStatusHistory `json:"status_history,omitempty" gorm:"foreignKey:InvoiceID"`
}

// PartnerInvoiceStatusHistory keeps an append-only audit trail of invoice status
// transitions so an issued invoice can always be reviewed after the fact.
type PartnerInvoiceStatusHistory struct {
	ID         uuid.UUID  `json:"id" gorm:"type:uuid;primaryKey"`
	InvoiceID  uuid.UUID  `json:"invoice_id" gorm:"type:uuid;not null;index"`
	FromStatus string     `json:"from_status" gorm:"type:varchar(30)"`
	ToStatus   string     `json:"to_status" gorm:"type:varchar(30);not null"`
	Note       string     `json:"note" gorm:"type:text"`
	ChangedBy  *uuid.UUID `json:"changed_by" gorm:"type:uuid"`
	CreatedAt  time.Time  `json:"created_at"`
}

func (p *PartnerInvoiceStatusHistory) BeforeCreate(_ *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}

func (p *PartnerInvoice) BeforeCreate(_ *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}

type PartnerInvoiceLine struct {
	ID          uuid.UUID `json:"id" gorm:"type:uuid;primaryKey"`
	InvoiceID   uuid.UUID `json:"invoice_id" gorm:"type:uuid;not null;index"`
	Description string    `json:"description" gorm:"type:text;not null"`
	LineType    string    `json:"line_type" gorm:"type:varchar(20);not null"`
	Quantity    float64   `json:"quantity" gorm:"not null"`
	Unit        string    `json:"unit" gorm:"type:varchar(30)"`
	UnitPrice   int       `json:"unit_price"`
	TaxAmount   int       `json:"tax_amount"`
	Total       int       `json:"total"`
	// Service period (bandwidth) and the source reference that produced the
	// line. The composite unique index prevents the same source (a service
	// period or an inventory allocation) from being billed twice.
	PeriodYear    *int       `json:"period_year" gorm:"uniqueIndex:idx_partner_invoice_line_source"`
	PeriodMonth   *int       `json:"period_month" gorm:"uniqueIndex:idx_partner_invoice_line_source"`
	SourceRefType string     `json:"source_ref_type" gorm:"type:varchar(30);uniqueIndex:idx_partner_invoice_line_source"`
	SourceRefID   *uuid.UUID `json:"source_ref_id" gorm:"type:uuid;uniqueIndex:idx_partner_invoice_line_source"`
}

func (p *PartnerInvoiceLine) BeforeCreate(_ *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}

type PartnerPayment struct {
	ID              uuid.UUID  `json:"id" gorm:"type:uuid;primaryKey"`
	InvoiceID       uuid.UUID  `json:"invoice_id" gorm:"type:uuid;not null;index"`
	SubmittedByID   *uuid.UUID `json:"submitted_by_id" gorm:"type:uuid"`
	SubmittedAt     time.Time  `json:"submitted_at"`
	Amount          int        `json:"amount" gorm:"not null"`
	Method          string     `json:"method" gorm:"type:varchar(50)"`
	Reference       string     `json:"reference" gorm:"type:varchar(255)"`
	ProofURL        string     `json:"proof_url" gorm:"type:text"`
	Status          string     `json:"status" gorm:"type:varchar(20);not null;default:pending;index"`
	VerifiedByID    *uuid.UUID `json:"verified_by_id" gorm:"type:uuid"`
	VerifiedAt      *time.Time `json:"verified_at"`
	RejectionReason string     `json:"rejection_reason" gorm:"type:text"`
	// IdempotencyKey makes payment submission safe to retry: repeating the same
	// key returns the original submission instead of creating a duplicate.
	IdempotencyKey string         `json:"idempotency_key" gorm:"type:varchar(120);uniqueIndex"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `json:"deleted_at" gorm:"index"`
}

func (p *PartnerPayment) BeforeCreate(_ *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}

type PartnerInvoiceNotification struct {
	ID             uuid.UUID `json:"id" gorm:"type:uuid;primaryKey"`
	InvoiceID      uuid.UUID `json:"invoice_id" gorm:"type:uuid;not null;index"`
	Recipient      string    `json:"recipient" gorm:"type:varchar(255);not null"`
	Kind           string    `json:"kind" gorm:"type:varchar(30);not null"`
	Status         string    `json:"status" gorm:"type:varchar(30);not null"`
	BrevoMessageID string    `json:"brevo_message_id" gorm:"type:varchar(255);index"`
	ErrorMessage   string    `json:"error_message" gorm:"type:text"`
	// IdempotencyKey keeps a single job from sending duplicate emails. Resends
	// intentionally use a new key and therefore produce a new audit row.
	IdempotencyKey string    `json:"idempotency_key" gorm:"type:varchar(160);uniqueIndex"`
	CreatedAt      time.Time `json:"created_at"`
}

func (p *PartnerInvoiceNotification) BeforeCreate(_ *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}

// PartnerBandwidthUsage is a metered reading for a usage-billed service. Staff
// validates the number before it becomes an invoice line; the combination of
// service and period is unique so a period cannot be billed twice.
type PartnerBandwidthUsage struct {
	ID           uuid.UUID                `json:"id" gorm:"type:uuid;primaryKey"`
	PartnerID    uuid.UUID                `json:"partner_id" gorm:"type:uuid;not null;index"`
	ServiceID    uuid.UUID                `json:"service_id" gorm:"type:uuid;not null;uniqueIndex:idx_partner_usage_period"`
	PeriodYear   int                      `json:"period_year" gorm:"not null;uniqueIndex:idx_partner_usage_period"`
	PeriodMonth  int                      `json:"period_month" gorm:"not null;uniqueIndex:idx_partner_usage_period"`
	Quantity     float64                  `json:"quantity" gorm:"not null"`
	Unit         string                   `json:"unit" gorm:"type:varchar(30)"`
	DataSource   string                   `json:"data_source" gorm:"type:varchar(50)"` // manual, meter, import
	EvidenceURL  string                   `json:"evidence_url" gorm:"type:text"`
	Notes        string                   `json:"notes" gorm:"type:text"`
	Status       string                   `json:"status" gorm:"type:varchar(20);not null;default:pending;index"` // pending, approved, rejected
	ApprovedByID *uuid.UUID               `json:"approved_by_id" gorm:"type:uuid"`
	ApprovedAt   *time.Time               `json:"approved_at"`
	RejectReason string                   `json:"reject_reason" gorm:"type:text"`
	CreatedByID  *uuid.UUID               `json:"created_by_id" gorm:"type:uuid"`
	CreatedAt    time.Time                `json:"created_at"`
	UpdatedAt    time.Time                `json:"updated_at"`
	DeletedAt    gorm.DeletedAt           `json:"deleted_at" gorm:"index"`
	Service      *PartnerBandwidthService `json:"service,omitempty" gorm:"foreignKey:ServiceID"`
	// PartnerName/PartnerCode are filled for cross-partner listings only and are
	// never persisted.
	PartnerName string `json:"partner_name,omitempty" gorm:"-"`
	PartnerCode string `json:"partner_code,omitempty" gorm:"-"`
}

func (p *PartnerBandwidthUsage) BeforeCreate(_ *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}

// PartnerInventoryAllocation records items handed to a partner. A loan stays a
// Cantika.net asset and can be returned; a sale reduces stock exactly once and
// must be linked to a partner invoice line so it is billed.
type PartnerInventoryAllocation struct {
	ID              uuid.UUID            `json:"id" gorm:"type:uuid;primaryKey"`
	AllocationNo    string               `json:"allocation_no" gorm:"type:varchar(60);uniqueIndex"`
	PartnerID       uuid.UUID            `json:"partner_id" gorm:"type:uuid;not null;index"`
	ItemID          uuid.UUID            `json:"item_id" gorm:"type:uuid;not null;index"`
	SerialItemID    *uuid.UUID           `json:"serial_item_id" gorm:"type:uuid;index"`
	Quantity        float64              `json:"quantity" gorm:"not null"`
	UnitPrice       int                  `json:"unit_price"`
	FromLocationID  *uuid.UUID           `json:"from_location_id" gorm:"type:uuid;index"`
	TransactionType string               `json:"transaction_type" gorm:"type:varchar(20);not null;index"`      // loan, sale
	Status          string               `json:"status" gorm:"type:varchar(20);not null;default:active;index"` // active, returned, cancelled
	AllocatedAt     time.Time            `json:"allocated_at"`
	ReturnedAt      *time.Time           `json:"returned_at"`
	InvoiceLineID   *uuid.UUID           `json:"invoice_line_id" gorm:"type:uuid;index"`
	Notes           string               `json:"notes" gorm:"type:text"`
	CreatedByID     *uuid.UUID           `json:"created_by_id" gorm:"type:uuid"`
	CreatedAt       time.Time            `json:"created_at"`
	UpdatedAt       time.Time            `json:"updated_at"`
	DeletedAt       gorm.DeletedAt       `json:"deleted_at" gorm:"index"`
	Item            *InventoryItem       `json:"item,omitempty" gorm:"foreignKey:ItemID"`
	SerialItem      *InventorySerialItem `json:"serial_item,omitempty" gorm:"foreignKey:SerialItemID"`
	// PartnerName/PartnerCode are filled for cross-partner listings only and are
	// never persisted.
	PartnerName string `json:"partner_name,omitempty" gorm:"-"`
	PartnerCode string `json:"partner_code,omitempty" gorm:"-"`
}

func (p *PartnerInventoryAllocation) BeforeCreate(_ *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}
