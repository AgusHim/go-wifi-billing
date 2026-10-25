package services

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/Agushim/go_wifi_billing/dto"
	"github.com/Agushim/go_wifi_billing/models"
	"github.com/google/uuid"
)

// ── Bandwidth usage ────────────────────────────────────────────────────────

// RecordUsage stores a metered reading for a usage-billed service. The reading
// starts as `pending` and only an approved reading can be billed.
func (s *partnerService) RecordUsage(partnerID uuid.UUID, in dto.PartnerBandwidthUsageRequest, actorID uuid.UUID) (*models.PartnerBandwidthUsage, error) {
	serviceID, err := uuid.Parse(strings.TrimSpace(in.ServiceID))
	if err != nil {
		return nil, errors.New("invalid service_id")
	}
	service, err := s.repo.FindService(serviceID)
	if err != nil || service.PartnerID != partnerID {
		return nil, errors.New("bandwidth service not found for this partner")
	}
	if service.BillingMode != "usage" {
		return nil, errors.New("usage can only be recorded for usage-billed services")
	}
	if in.PeriodYear < 2000 || in.PeriodYear > 2200 {
		return nil, errors.New("period_year is invalid")
	}
	if in.PeriodMonth < 1 || in.PeriodMonth > 12 {
		return nil, errors.New("period_month must be between 1 and 12")
	}
	if in.Quantity < 0 {
		return nil, errors.New("quantity must not be negative")
	}
	if _, err := s.repo.FindUsageByPeriod(serviceID, in.PeriodYear, in.PeriodMonth); err == nil {
		return nil, errors.New("usage for this service period already exists")
	}
	source := strings.TrimSpace(in.DataSource)
	if source == "" {
		source = "manual"
	}
	row := &models.PartnerBandwidthUsage{
		PartnerID: partnerID, ServiceID: serviceID, PeriodYear: in.PeriodYear, PeriodMonth: in.PeriodMonth,
		Quantity: in.Quantity, Unit: strings.TrimSpace(in.Unit), DataSource: source,
		EvidenceURL: strings.TrimSpace(in.EvidenceURL), Notes: strings.TrimSpace(in.Notes),
		Status: "pending", CreatedByID: &actorID,
	}
	if err := s.repo.CreateUsage(row); err != nil {
		return nil, err
	}
	return row, nil
}

func (s *partnerService) ReviewUsage(id uuid.UUID, in dto.PartnerBandwidthUsageApprovalRequest, actorID uuid.UUID) (*models.PartnerBandwidthUsage, error) {
	row, err := s.repo.FindUsage(id)
	if err != nil {
		return nil, errors.New("usage not found")
	}
	if row.Status != "pending" {
		return nil, errors.New("usage has already been reviewed")
	}
	now := time.Now()
	row.ApprovedByID = &actorID
	row.ApprovedAt = &now
	if in.Approve {
		row.Status = "approved"
		row.RejectReason = ""
	} else {
		if strings.TrimSpace(in.RejectReason) == "" {
			return nil, errors.New("reject_reason is required")
		}
		row.Status = "rejected"
		row.RejectReason = strings.TrimSpace(in.RejectReason)
	}
	if err := s.repo.SaveUsage(row); err != nil {
		return nil, err
	}
	return row, nil
}

func (s *partnerService) ListUsages(partnerID *uuid.UUID, serviceID *uuid.UUID, status string) ([]models.PartnerBandwidthUsage, error) {
	return s.repo.ListUsages(partnerID, serviceID, status)
}

func periodLabel(year, month int) string {
	names := []string{"Januari", "Februari", "Maret", "April", "Mei", "Juni", "Juli", "Agustus", "September", "Oktober", "November", "Desember"}
	label := fmt.Sprintf("%02d", month)
	if month >= 1 && month <= 12 {
		label = names[month-1]
	}
	return fmt.Sprintf("%s %d", label, year)
}

// ── Invoice generation from a service ──────────────────────────────────────

// GenerateInvoiceForService builds a draft invoice for one service period.
// Fixed billing uses the contract rate, usage billing uses the approved
// reading, and manual billing uses the amount entered by staff. The period is
// recorded as the line source so the same period cannot be billed twice.
func (s *partnerService) GenerateInvoiceForService(partnerID uuid.UUID, in dto.PartnerInvoiceFromServiceRequest, actorID uuid.UUID) (*models.PartnerInvoice, error) {
	serviceID, err := uuid.Parse(strings.TrimSpace(in.ServiceID))
	if err != nil {
		return nil, errors.New("invalid service_id")
	}
	service, err := s.repo.FindService(serviceID)
	if err != nil || service.PartnerID != partnerID {
		return nil, errors.New("bandwidth service not found for this partner")
	}
	if in.PeriodYear < 2000 || in.PeriodYear > 2200 || in.PeriodMonth < 1 || in.PeriodMonth > 12 {
		return nil, errors.New("a valid period_year and period_month are required")
	}
	if _, err := s.repo.FindInvoiceLineBySource("bandwidth", serviceID, &in.PeriodYear, &in.PeriodMonth); err == nil {
		return nil, errors.New("this service period has already been billed")
	}

	quantity := 1.0
	unit := "bulan"
	unitPrice := service.Rate
	description := fmt.Sprintf("%s - %s", service.Name, periodLabel(in.PeriodYear, in.PeriodMonth))
	switch service.BillingMode {
	case "fixed":
		// contract rate, one month
	case "usage":
		usage, err := s.repo.FindUsageByPeriod(serviceID, in.PeriodYear, in.PeriodMonth)
		if err != nil {
			return nil, errors.New("approved usage for this period is required before billing")
		}
		if usage.Status != "approved" {
			return nil, errors.New("usage for this period has not been approved")
		}
		quantity = usage.Quantity
		unit = "unit"
		if strings.TrimSpace(usage.Unit) != "" {
			unit = strings.TrimSpace(usage.Unit)
		}
		description = fmt.Sprintf("%s - %s (%s)", service.Name, periodLabel(in.PeriodYear, in.PeriodMonth), usage.DataSource)
	case "manual":
		if in.Amount <= 0 {
			return nil, errors.New("a positive amount is required for manual billing")
		}
		unitPrice = in.Amount
	default:
		return nil, errors.New("unsupported billing_mode")
	}
	if quantity <= 0 || unitPrice <= 0 {
		return nil, errors.New("billing produced a non-positive amount")
	}

	dueDate := time.Date(in.PeriodYear, time.Month(in.PeriodMonth)+1, 1, 0, 0, 0, 0, time.UTC)
	if service.DueDay > 0 {
		dueDate = dueDate.AddDate(0, 0, service.DueDay-1)
	}
	if in.DueDate != nil && !in.DueDate.IsZero() {
		dueDate = *in.DueDate
	}
	year, month := in.PeriodYear, in.PeriodMonth
	return s.CreateInvoice(dto.PartnerInvoiceRequest{
		PartnerID: partnerID.String(),
		DueDate:   dueDate,
		Lines: []dto.PartnerInvoiceLineRequest{{
			Description: description, LineType: "bandwidth", Quantity: quantity, Unit: unit,
			UnitPrice: unitPrice, TaxAmount: in.TaxAmount, ServiceID: serviceID.String(),
			PeriodYear: &year, PeriodMonth: &month,
		}},
	}, actorID)
}

// ── Invoice audit + notifications ──────────────────────────────────────────

func (s *partnerService) ListInvoiceHistory(invoiceID uuid.UUID) ([]models.PartnerInvoiceStatusHistory, error) {
	return s.repo.ListStatusHistory(invoiceID)
}

func (s *partnerService) ListInvoiceNotifications(invoiceID uuid.UUID) ([]models.PartnerInvoiceNotification, error) {
	return s.repo.ListNotifications(invoiceID)
}

// ResendInvoiceNotification always creates a new audit row with a distinct
// idempotency key so a manual resend is visible and never silently dropped.
func (s *partnerService) ResendInvoiceNotification(invoiceID uuid.UUID, actorID uuid.UUID) (*models.PartnerInvoiceNotification, error) {
	invoice, err := s.repo.FindInvoice(invoiceID)
	if err != nil {
		return nil, errors.New("invoice not found")
	}
	if invoice.Status == "draft" {
		return nil, errors.New("a draft invoice cannot be sent")
	}
	kind := "invoice"
	if invoice.BalanceDue > 0 && invoice.Status != "paid" {
		kind = "reminder"
	}
	key := fmt.Sprintf("resend:%s:%d", invoice.ID.String(), time.Now().UnixNano())
	return s.sendInvoiceEmail(*invoice, kind, key), nil
}

// ── Inventory allocation ───────────────────────────────────────────────────

func (s *partnerService) CreateAllocation(partnerID uuid.UUID, in dto.PartnerAllocationRequest, actorID uuid.UUID) (*models.PartnerInventoryAllocation, error) {
	if _, err := s.repo.FindPartner(partnerID); err != nil {
		return nil, errors.New("partner not found")
	}
	itemID, err := uuid.Parse(strings.TrimSpace(in.ItemID))
	if err != nil {
		return nil, errors.New("invalid item_id")
	}
	transactionType := strings.TrimSpace(in.TransactionType)
	if transactionType != "loan" && transactionType != "sale" {
		return nil, errors.New("transaction_type must be loan or sale")
	}
	if strings.TrimSpace(in.FromLocationID) == "" {
		return nil, errors.New("from_location_id is required")
	}
	fromLocationID, err := uuid.Parse(strings.TrimSpace(in.FromLocationID))
	if err != nil {
		return nil, errors.New("invalid from_location_id")
	}
	if transactionType == "sale" && in.UnitPrice <= 0 {
		return nil, errors.New("a sale requires a positive unit_price")
	}
	row := &models.PartnerInventoryAllocation{
		AllocationNo: fmt.Sprintf("PA-%s-%s", time.Now().Format("20060102150405"), uuid.NewString()[:6]),
		PartnerID:    partnerID, ItemID: itemID, Quantity: in.Quantity, UnitPrice: in.UnitPrice,
		FromLocationID: &fromLocationID, TransactionType: transactionType, Status: "active",
		AllocatedAt: time.Now(), Notes: strings.TrimSpace(in.Notes), CreatedByID: &actorID,
	}
	if strings.TrimSpace(in.SerialItemID) != "" {
		serialID, err := uuid.Parse(strings.TrimSpace(in.SerialItemID))
		if err != nil {
			return nil, errors.New("invalid serial_item_id")
		}
		row.SerialItemID = &serialID
	}
	if err := s.repo.CreateAllocationWithStock(row); err != nil {
		return nil, err
	}
	return row, nil
}

func (s *partnerService) ListAllocations(partnerID *uuid.UUID, status, transactionType string) ([]models.PartnerInventoryAllocation, error) {
	return s.repo.ListAllocations(partnerID, status, transactionType)
}

func (s *partnerService) ReturnAllocation(id uuid.UUID, in dto.PartnerAllocationReturnRequest, actorID uuid.UUID) (*models.PartnerInventoryAllocation, error) {
	allocation, err := s.repo.FindAllocation(id)
	if err != nil {
		return nil, errors.New("allocation not found")
	}
	var toLocationID uuid.UUID
	if strings.TrimSpace(in.ToLocationID) != "" {
		parsed, err := uuid.Parse(strings.TrimSpace(in.ToLocationID))
		if err != nil {
			return nil, errors.New("invalid to_location_id")
		}
		toLocationID = parsed
	} else if allocation.FromLocationID != nil {
		toLocationID = *allocation.FromLocationID
	} else {
		return nil, errors.New("to_location_id is required")
	}
	if err := s.repo.ReturnAllocation(allocation, toLocationID, strings.TrimSpace(in.Notes)); err != nil {
		return nil, err
	}
	return allocation, nil
}

// ── Receivables aging, reminders, overdue ──────────────────────────────────

// AgingReport groups outstanding receivables per partner into age buckets.
// When partnerID is set the report is scoped to a single partner (portal use).
func (s *partnerService) AgingReport(partnerID *uuid.UUID) ([]PartnerAgingRow, error) {
	invoices, err := s.repo.ListInvoices(partnerID)
	if err != nil {
		return nil, err
	}
	rows := map[uuid.UUID]*PartnerAgingRow{}
	order := []uuid.UUID{}
	now := time.Now()
	for i := range invoices {
		invoice := invoices[i]
		if invoice.BalanceDue <= 0 || invoice.Status == "draft" || invoice.Status == "void" {
			continue
		}
		row, ok := rows[invoice.PartnerID]
		if !ok {
			name := invoice.PartnerNameSnapshot
			if invoice.Partner.ID != uuid.Nil && invoice.Partner.Name != "" {
				name = invoice.Partner.Name
			}
			row = &PartnerAgingRow{PartnerID: invoice.PartnerID, PartnerName: name}
			rows[invoice.PartnerID] = row
			order = append(order, invoice.PartnerID)
		}
		row.InvoiceCount++
		row.TotalDue += invoice.BalanceDue
		days := int(now.Sub(invoice.DueDate).Hours() / 24)
		switch {
		case days <= 0:
			row.Current += invoice.BalanceDue
		case days <= 30:
			row.Days1To30 += invoice.BalanceDue
		case days <= 60:
			row.Days31To60 += invoice.BalanceDue
		case days <= 90:
			row.Days61To90 += invoice.BalanceDue
		default:
			row.Over90 += invoice.BalanceDue
		}
	}
	result := make([]PartnerAgingRow, 0, len(order))
	for _, id := range order {
		result = append(result, *rows[id])
	}
	return result, nil
}

// SendDueReminders emails partners whose invoices are due soon or overdue.
// Each invoice+day gets one idempotency key, so running the job repeatedly on
// the same day never produces duplicate emails.
func (s *partnerService) SendDueReminders() (int, error) {
	invoices, err := s.repo.ListInvoicesDueForReminder()
	if err != nil {
		return 0, err
	}
	sent := 0
	for i := range invoices {
		invoice := invoices[i]
		key := fmt.Sprintf("reminder:%s:%s", invoice.ID.String(), time.Now().Format("20060102"))
		if _, err := s.repo.FindNotificationByIdempotencyKey(key); err == nil {
			continue
		}
		recipient := invoice.PartnerEmailSnapshot
		if invoice.Partner.BillingEmail != "" {
			recipient = invoice.Partner.BillingEmail
		}
		invoice.PartnerEmailSnapshot = recipient
		s.sendInvoiceEmail(invoice, "reminder", key)
		sent++
	}
	return sent, nil
}

// MarkOverdueInvoices flags invoices past their due date. It is idempotent and
// records a status-history entry per transition.
func (s *partnerService) MarkOverdueInvoices() (int, error) {
	invoices, err := s.repo.ListOverdueInvoices()
	if err != nil {
		return 0, err
	}
	updated := 0
	for i := range invoices {
		invoice := invoices[i]
		previous := invoice.Status
		invoice.Status = "overdue"
		if err := s.repo.SaveInvoice(&invoice); err != nil {
			return updated, err
		}
		s.recordStatus(invoice.ID, previous, "overdue", "due date passed", nil)
		updated++
	}
	return updated, nil
}

// GenerateRecurringDrafts creates draft invoices for active fixed-rate services
// for the current period when that period has not been billed yet. Drafts still
// need staff review before they are issued, so nothing is sent automatically.
func (s *partnerService) GenerateRecurringDrafts() (int, error) {
	services, err := s.repo.ListServicesDueForBilling()
	if err != nil {
		return 0, err
	}
	now := time.Now()
	year, month := now.Year(), int(now.Month())
	created := 0
	for i := range services {
		service := services[i]
		if _, err := s.repo.FindInvoiceLineBySource("bandwidth", service.ID, &year, &month); err == nil {
			continue // this period is already billed
		}
		if _, err := s.GenerateInvoiceForService(service.PartnerID, dto.PartnerInvoiceFromServiceRequest{
			ServiceID: service.ID.String(), PeriodYear: year, PeriodMonth: month,
		}, uuid.Nil); err != nil {
			continue
		}
		created++
	}
	return created, nil
}

// StartScheduler periodically marks overdue invoices, sends due reminders, and
// drafts recurring invoices. It is opt-in through an environment flag.
func (s *partnerService) StartScheduler() {
	if !partnerAutomationEnabled() {
		return
	}
	interval := resolvePartnerAutomationInterval()
	go func() {
		s.runAutomationCycle("initial")
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			s.runAutomationCycle("scheduled")
		}
	}()
}

func (s *partnerService) runAutomationCycle(phase string) {
	if updated, err := s.MarkOverdueInvoices(); err != nil {
		log.Printf("[partner-automation] %s overdue run failed: %v", phase, err)
	} else if updated > 0 {
		log.Printf("[partner-automation] %s run marked %d invoices overdue", phase, updated)
	}
	if sent, err := s.SendDueReminders(); err != nil {
		log.Printf("[partner-automation] %s reminder run failed: %v", phase, err)
	} else if sent > 0 {
		log.Printf("[partner-automation] %s run processed %d reminders", phase, sent)
	}
	if created, err := s.GenerateRecurringDrafts(); err != nil {
		log.Printf("[partner-automation] %s draft run failed: %v", phase, err)
	} else if created > 0 {
		log.Printf("[partner-automation] %s run created %d draft invoices", phase, created)
	}
}

func partnerAutomationEnabled() bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv("FEATURE_PARTNER_AUTOMATION_ENABLED")))
	return value == "1" || value == "true" || value == "yes"
}

func resolvePartnerAutomationInterval() time.Duration {
	value := strings.TrimSpace(os.Getenv("PARTNER_AUTOMATION_INTERVAL"))
	if value == "" {
		return 6 * time.Hour
	}
	interval, err := time.ParseDuration(value)
	if err != nil || interval <= 0 {
		return 6 * time.Hour
	}
	return interval
}

// ── Legal documents ────────────────────────────────────────────────────────

func (s *partnerService) CreateLegalDocument(in dto.LegalDocumentRequest, actorID uuid.UUID) (*models.LegalDocument, error) {
	if strings.TrimSpace(in.Name) == "" || strings.TrimSpace(in.FilePath) == "" {
		return nil, errors.New("name and file_path are required")
	}
	status := strings.TrimSpace(in.Status)
	if status == "" {
		status = "draft"
	}
	if status != "draft" && status != "published" && status != "archived" {
		return nil, errors.New("status must be draft, published, or archived")
	}
	row := &models.LegalDocument{
		Name: strings.TrimSpace(in.Name), Category: strings.TrimSpace(in.Category), Version: strings.TrimSpace(in.Version),
		EffectiveDate: in.EffectiveDate, ExpiresAt: in.ExpiresAt, FilePath: strings.TrimSpace(in.FilePath),
		MimeType: strings.TrimSpace(in.MimeType), FileSize: in.FileSize, Status: status,
		IsPartnerVisible: in.IsPartnerVisible, Notes: strings.TrimSpace(in.Notes), UploadedByID: &actorID,
	}
	if err := s.repo.CreateDocument(row); err != nil {
		return nil, err
	}
	return row, nil
}

func (s *partnerService) UpdateLegalDocument(id uuid.UUID, in dto.LegalDocumentRequest) (*models.LegalDocument, error) {
	row, err := s.repo.FindDocument(id)
	if err != nil {
		return nil, errors.New("document not found")
	}
	if name := strings.TrimSpace(in.Name); name != "" {
		row.Name = name
	}
	if path := strings.TrimSpace(in.FilePath); path != "" {
		row.FilePath = path
	}
	if status := strings.TrimSpace(in.Status); status != "" {
		if status != "draft" && status != "published" && status != "archived" {
			return nil, errors.New("status must be draft, published, or archived")
		}
		row.Status = status
	}
	row.Category = strings.TrimSpace(in.Category)
	row.Version = strings.TrimSpace(in.Version)
	row.MimeType = strings.TrimSpace(in.MimeType)
	if in.EffectiveDate != nil {
		row.EffectiveDate = in.EffectiveDate
	}
	row.ExpiresAt = in.ExpiresAt
	if in.FileSize > 0 {
		row.FileSize = in.FileSize
	}
	row.IsPartnerVisible = in.IsPartnerVisible
	row.Notes = strings.TrimSpace(in.Notes)
	if err := s.repo.SaveDocument(row); err != nil {
		return nil, err
	}
	return row, nil
}

func (s *partnerService) ListLegalDocuments(partnerVisible *bool, category string) ([]models.LegalDocument, error) {
	return s.repo.ListDocuments(partnerVisible, category)
}

// GetLegalDocument enforces partner scoping: a partner user may only read a
// document that is published and explicitly shared with partners.
func (s *partnerService) GetLegalDocument(id uuid.UUID, partnerID *uuid.UUID) (*models.LegalDocument, error) {
	row, err := s.repo.FindDocument(id)
	if err != nil {
		return nil, errors.New("document not found")
	}
	if partnerID != nil && (!row.IsPartnerVisible || row.Status != "published") {
		return nil, errors.New("document not found")
	}
	return row, nil
}

func (s *partnerService) DeleteLegalDocument(id uuid.UUID) error {
	if _, err := s.repo.FindDocument(id); err != nil {
		return errors.New("document not found")
	}
	return s.repo.DeleteDocument(id)
}

// ── Brevo webhook ──────────────────────────────────────────────────────────

// RecordBrevoEvent attaches a delivery/bounce/complaint event to the matching
// notification. Events never change invoice or payment state; they only add
// staff-visible context.
func (s *partnerService) RecordBrevoEvent(in dto.BrevoWebhookEvent) error {
	messageID := strings.TrimSpace(in.MessageID)
	if messageID == "" {
		return errors.New("message-id is required")
	}
	note, err := s.repo.FindNotificationByMessageID(messageID)
	if err != nil {
		return errors.New("notification not found for this message id")
	}
	event := strings.TrimSpace(in.Event)
	if event == "" {
		return errors.New("event is required")
	}
	detail := event
	if strings.TrimSpace(in.Reason) != "" {
		detail = fmt.Sprintf("%s: %s", event, strings.TrimSpace(in.Reason))
	}
	if strings.TrimSpace(note.ErrorMessage) == "" {
		note.ErrorMessage = detail
	} else {
		note.ErrorMessage = note.ErrorMessage + " | " + detail
	}
	if event == "delivered" {
		note.Status = "delivered"
	}
	if event == "bounce" || event == "blocked" || event == "complaint" {
		note.Status = event
	}
	return s.repo.SaveNotification(note)
}
