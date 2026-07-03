package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ChartOfAccount struct {
	ID        uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	Code      string         `json:"code" gorm:"uniqueIndex;not null"`
	Name      string         `json:"name" gorm:"not null"`
	Type      string         `json:"type" gorm:"index"`
	IsActive  bool           `json:"is_active" gorm:"default:true"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index"`
}

func (c *ChartOfAccount) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}

type AccountingJournal struct {
	ID            uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	JournalNumber string         `json:"journal_number" gorm:"uniqueIndex;not null"`
	JournalDate   time.Time      `json:"journal_date" gorm:"index"`
	SourceType    string         `json:"source_type" gorm:"uniqueIndex:idx_accounting_journal_source"`
	SourceID      *uuid.UUID     `json:"source_id" gorm:"type:uuid;uniqueIndex:idx_accounting_journal_source"`
	Description   string         `json:"description"`
	Status        string         `json:"status" gorm:"index"`
	PostedBy      *uuid.UUID     `json:"posted_by" gorm:"type:uuid"`
	PostedAt      *time.Time     `json:"posted_at"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `json:"deleted_at" gorm:"index"`

	Lines  []AccountingJournalLine `json:"lines,omitempty" gorm:"foreignKey:JournalID"`
	Poster *User                   `json:"poster,omitempty" gorm:"foreignKey:PostedBy"`
}

func (a *AccountingJournal) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}

type AccountingJournalLine struct {
	ID         uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	JournalID  uuid.UUID      `json:"journal_id" gorm:"type:uuid;not null;index"`
	AccountID  uuid.UUID      `json:"account_id" gorm:"type:uuid;not null;index"`
	Debit      float64        `json:"debit"`
	Credit     float64        `json:"credit"`
	EntityType string         `json:"entity_type"`
	EntityID   *uuid.UUID     `json:"entity_id" gorm:"type:uuid;index"`
	Memo       string         `json:"memo"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `json:"deleted_at" gorm:"index"`

	Journal *AccountingJournal `json:"journal,omitempty" gorm:"foreignKey:JournalID"`
	Account *ChartOfAccount    `json:"account,omitempty" gorm:"foreignKey:AccountID"`
}

func (a *AccountingJournalLine) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}

type AccountingPeriodLock struct {
	ID        uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	Period    string         `json:"period" gorm:"uniqueIndex;not null"`
	IsLocked  bool           `json:"is_locked" gorm:"default:true"`
	LockedBy  *uuid.UUID     `json:"locked_by" gorm:"type:uuid"`
	LockedAt  *time.Time     `json:"locked_at"`
	Notes     string         `json:"notes"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index"`

	Locker *User `json:"locker,omitempty" gorm:"foreignKey:LockedBy"`
}

func (a *AccountingPeriodLock) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}

type SupplierInvoice struct {
	ID                uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	SupplierID        uuid.UUID      `json:"supplier_id" gorm:"type:uuid;not null;index"`
	GoodsReceiptID    *uuid.UUID     `json:"goods_receipt_id" gorm:"type:uuid;uniqueIndex"`
	InvoiceNumber     string         `json:"invoice_number" gorm:"uniqueIndex;not null"`
	InvoiceDate       time.Time      `json:"invoice_date" gorm:"index"`
	DueDate           *time.Time     `json:"due_date"`
	Status            string         `json:"status" gorm:"index"`
	Subtotal          float64        `json:"subtotal"`
	Tax               float64        `json:"tax"`
	Discount          float64        `json:"discount"`
	GrandTotal        float64        `json:"grand_total"`
	PaidAmount        float64        `json:"paid_amount"`
	OutstandingAmount float64        `json:"outstanding_amount"`
	Notes             string         `json:"notes"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `json:"deleted_at" gorm:"index"`

	Supplier     *Supplier         `json:"supplier,omitempty" gorm:"foreignKey:SupplierID"`
	GoodsReceipt *GoodsReceipt     `json:"goods_receipt,omitempty" gorm:"foreignKey:GoodsReceiptID"`
	Payments     []SupplierPayment `json:"payments,omitempty" gorm:"foreignKey:SupplierInvoiceID"`
}

func (s *SupplierInvoice) BeforeCreate(tx *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}

type SupplierPayment struct {
	ID                uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	SupplierInvoiceID uuid.UUID      `json:"supplier_invoice_id" gorm:"type:uuid;not null;index"`
	SupplierID        uuid.UUID      `json:"supplier_id" gorm:"type:uuid;not null;index"`
	PaymentNumber     string         `json:"payment_number" gorm:"uniqueIndex;not null"`
	PaymentDate       time.Time      `json:"payment_date" gorm:"index"`
	Amount            float64        `json:"amount"`
	PaymentMethod     string         `json:"payment_method"`
	CashAccountCode   string         `json:"cash_account_code"`
	ReferenceNumber   string         `json:"reference_number"`
	Notes             string         `json:"notes"`
	CreatedBy         *uuid.UUID     `json:"created_by" gorm:"type:uuid"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `json:"deleted_at" gorm:"index"`

	SupplierInvoice *SupplierInvoice `json:"supplier_invoice,omitempty" gorm:"foreignKey:SupplierInvoiceID"`
	Supplier        *Supplier        `json:"supplier,omitempty" gorm:"foreignKey:SupplierID"`
	Creator         *User            `json:"creator,omitempty" gorm:"foreignKey:CreatedBy"`
}

func (s *SupplierPayment) BeforeCreate(tx *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}
