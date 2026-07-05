package services

import (
	"crypto/rand"
	"errors"
	"fmt"
	"log"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/Agushim/go_wifi_billing/models"
	"github.com/Agushim/go_wifi_billing/repositories"
	"github.com/Agushim/go_wifi_billing/utils"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BillService interface {
	GetAll(page, limit int, search string, adminID string, status string, startAt string, endAt string, coverageIDs []string) ([]models.Bill, int64, error)
	GetByID(id string) (models.Bill, error)
	Create(input models.Bill) (models.Bill, error)
	Update(id string, input models.Bill) (models.Bill, error)
	Delete(id string) error
	DeleteCurrentMonthUnpaidBills() (int64, error)
	GenerateMonthlyBills() error
	GenerateMonthlyBillsForPeriod(period string) (*BillGenerationResult, error)
	PreviewMonthlyBills(period string) (*BillGenerationResult, error)
	MarkOverdueBills(referenceTime time.Time, limit int) (int, error)
	StartOverdueScheduler()
	GetByPublicID(publicID string) (*models.Bill, error)
	GetByUserID(userID string) ([]models.Bill, error)
	GetUnpaidBills() ([]models.Bill, error)
	SendReminders() (map[string]interface{}, error)
	GetDashboardStats(month, year int, adminID string) (map[string]interface{}, error)
	GetDashboardCharts(months int, adminID string) (map[string]interface{}, error)
	GetRecentPaidBills(limit int) ([]models.Bill, error)
}

type BillGenerationResult struct {
	Period             string               `json:"period"`
	PeriodYear         int                  `json:"period_year"`
	PeriodMonth        int                  `json:"period_month"`
	DryRun             bool                 `json:"dry_run"`
	TotalCandidates    int                  `json:"total_candidates"`
	Eligible           int                  `json:"eligible"`
	WouldCreate        int                  `json:"would_create"`
	Created            int                  `json:"created"`
	SkippedExisting    int                  `json:"skipped_existing"`
	SkippedOutOfPeriod int                  `json:"skipped_out_of_period"`
	SkippedInvalid     int                  `json:"skipped_invalid"`
	EstimatedAmount    int                  `json:"estimated_amount"`
	Items              []BillGenerationItem `json:"items"`
}

type BillGenerationItem struct {
	SubscriptionID uuid.UUID `json:"subscription_id"`
	CustomerID     uuid.UUID `json:"customer_id"`
	CustomerName   string    `json:"customer_name"`
	PackageName    string    `json:"package_name"`
	Action         string    `json:"action"`
	Reason         string    `json:"reason"`
	Amount         int       `json:"amount"`
	ExistingBillID string    `json:"existing_bill_id,omitempty"`
	PublicID       string    `json:"public_id,omitempty"`
}

type billService struct {
	repo                   repositories.BillRepository
	subRepo                repositories.SubscriptionRepository
	waSvc                  WhatsAppService
	billingProvisioningSvc BillingProvisioningService
}

func NewBillService(
	repo repositories.BillRepository,
	subRepo repositories.SubscriptionRepository,
	waSvc WhatsAppService,
	billingProvisioningSvc BillingProvisioningService,
) BillService {
	return &billService{
		repo:                   repo,
		subRepo:                subRepo,
		waSvc:                  waSvc,
		billingProvisioningSvc: billingProvisioningSvc,
	}
}

func (s *billService) GetAll(page, limit int, search string, adminID string, status string, startAt string, endAt string, coverageIDs []string) ([]models.Bill, int64, error) {
	adminID = strings.TrimSpace(adminID)
	status = strings.TrimSpace(strings.ToLower(status))
	startAt = strings.TrimSpace(startAt)
	endAt = strings.TrimSpace(endAt)

	var parsedAdminID *uuid.UUID
	if adminID != "" {
		uid, err := uuid.Parse(adminID)
		if err != nil {
			return nil, 0, errors.New("invalid admin_id")
		}
		parsedAdminID = &uid
	}

	if status != "" {
		switch status {
		case "paid", "unpaid", "overdue":
		default:
			return nil, 0, errors.New("invalid status")
		}
	}

	startDate, endDate, err := parseBillDateRange(startAt, endAt)
	if err != nil {
		return nil, 0, err
	}

	var parsedCoverageIDs []uuid.UUID
	for _, cid := range coverageIDs {
		cid = strings.TrimSpace(cid)
		if cid != "" {
			uid, err := uuid.Parse(cid)
			if err != nil {
				return nil, 0, errors.New("invalid coverage_id")
			}
			parsedCoverageIDs = append(parsedCoverageIDs, uid)
		}
	}

	return s.repo.FindAllPaginated(page, limit, search, parsedAdminID, status, startDate, endDate, parsedCoverageIDs)
}

func (s *billService) GetByID(id string) (models.Bill, error) {
	return s.repo.FindByID(id)
}

func (s *billService) Create(input models.Bill) (models.Bill, error) {
	input.ID = uuid.New()
	input.CreatedAt = time.Now()
	input.UpdatedAt = time.Now()
	ensureBillPeriod(&input, time.Local, "manual")

	// Snapshot data customer/package saat bill dibuat agar tetap utuh kalau
	// nantinya customer/user/package terhapus.
	if sub, err := s.subRepo.FindByID(input.SubscriptionID); err == nil && sub != nil {
		applyBillSnapshot(&input, sub)
	}

	err := s.repo.Create(&input)
	return input, err
}

// applyBillSnapshot mengisi field snapshot di bill dari subscription + customer + package + coverage.
// Aman dipanggil walau sebagian data nil; field yang sudah berisi tidak ditimpa kosong.
func applyBillSnapshot(bill *models.Bill, sub *models.Subscription) {
	if sub == nil {
		return
	}
	if sub.Package != nil {
		if bill.PackageName == "" {
			bill.PackageName = sub.Package.Name
		}
		if bill.PackagePrice == 0 {
			bill.PackagePrice = sub.Package.Price
		}
	}
	cust := sub.Customer
	if cust == nil {
		return
	}
	if bill.CustomerServiceNumber == "" {
		bill.CustomerServiceNumber = cust.ServiceNumber
	}
	if bill.CustomerAddress == "" {
		bill.CustomerAddress = cust.Address
	}
	if cust.Coverage != nil && bill.CoverageName == "" {
		bill.CoverageName = cust.Coverage.Name
	}
	if cust.User != nil {
		if bill.CustomerName == "" {
			bill.CustomerName = cust.User.Name
		}
		if bill.CustomerPhone == "" {
			bill.CustomerPhone = cust.User.Phone
		}
		if bill.CustomerEmail == "" {
			bill.CustomerEmail = cust.User.Email
		}
	}
}
func (s *billService) GetByPublicID(publicID string) (*models.Bill, error) {
	bill, err := s.repo.FindByPublicID(publicID)
	if err != nil {
		return nil, err
	}
	return bill, nil
}
func (s *billService) GetByUserID(userID string) ([]models.Bill, error) {
	bill, err := s.repo.FindByUserID(userID)
	if err != nil {
		return nil, err
	}
	return bill, nil
}
func (s *billService) Update(id string, input models.Bill) (models.Bill, error) {
	bill, err := s.repo.FindByID(id)
	if err != nil {
		return bill, err
	}
	previousStatus := strings.TrimSpace(strings.ToLower(bill.Status))
	bill.Amount = input.Amount
	bill.Status = input.Status
	bill.BillDate = input.BillDate
	bill.DueDate = input.DueDate
	bill.TerminatedDate = input.TerminatedDate
	bill.PeriodYear = input.PeriodYear
	bill.PeriodMonth = input.PeriodMonth
	bill.PeriodStart = input.PeriodStart
	bill.PeriodEnd = input.PeriodEnd
	if strings.TrimSpace(input.Source) != "" {
		bill.Source = input.Source
	}
	bill.StatusReason = input.StatusReason
	ensureBillPeriod(&bill, time.Local, "manual")
	bill.UpdatedAt = time.Now()
	err = s.repo.Update(&bill)
	if err != nil {
		return bill, err
	}

	if previousStatus != "overdue" && strings.EqualFold(strings.TrimSpace(bill.Status), "overdue") && s.billingProvisioningSvc != nil {
		subscription, subErr := s.subRepo.FindByID(bill.SubscriptionID)
		if subErr != nil {
			log.Printf("[billing-provisioning] failed to load subscription %s for overdue bill %s: %v", bill.SubscriptionID, bill.ID, subErr)
		} else {
			s.billingProvisioningSvc.HandleBillOverdue(&bill, subscription)
		}
	}

	return bill, err
}

func (s *billService) Delete(id string) error {
	return s.repo.Delete(id)
}

func (s *billService) DeleteCurrentMonthUnpaidBills() (int64, error) {
	now := time.Now()
	startOfMonth, endOfMonth := billMonthRange(now.Year(), int(now.Month()))

	deleted, err := s.repo.DeleteUnpaidByBillDateRange(startOfMonth, endOfMonth)
	if err != nil {
		return 0, fmt.Errorf("failed to delete current month unpaid bills: %w", err)
	}

	return deleted, nil
}

func (s *billService) GenerateMonthlyBills() error {
	_, err := s.GenerateMonthlyBillsForPeriod("")
	return err
}

func (s *billService) GenerateMonthlyBillsForPeriod(period string) (*BillGenerationResult, error) {
	return s.runMonthlyBillGeneration(period, false)
}

func (s *billService) PreviewMonthlyBills(period string) (*BillGenerationResult, error) {
	return s.runMonthlyBillGeneration(period, true)
}

func (s *billService) runMonthlyBillGeneration(period string, dryRun bool) (*BillGenerationResult, error) {
	status := "active"
	// Luluskan false agar mengambil SEMUA active subscription, bukan hanya yang end_date-nya bulan ini.
	// Hal ini untuk memastikan customer yang telat bayar bulan lalu tetap mendapat tagihan baru.
	subs, err := s.subRepo.FindForBill(nil, &status, false)
	log.Printf("Found %d active subscriptions", len(subs))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch subscriptions: %w", err)
	}

	// Pastikan kita menggunakan zona waktu Indonesia (WIB)
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err == nil {
	} else {
		// Fallback ke UTC+7 jika tzdata tidak ada di sistem
		loc = time.FixedZone("WIB", 7*60*60)
	}
	now := time.Now().In(loc)

	billMonthStart, currentYear, currentMonth, err := resolveBillingPeriod(period, now, loc)
	if err != nil {
		return nil, err
	}
	periodEnd := billMonthStart.AddDate(0, 1, 0).Add(-time.Second)

	result := &BillGenerationResult{
		Period:          fmt.Sprintf("%04d-%02d", currentYear, currentMonth),
		PeriodYear:      currentYear,
		PeriodMonth:     currentMonth,
		DryRun:          dryRun,
		TotalCandidates: len(subs),
		Items:           make([]BillGenerationItem, 0, len(subs)),
	}

	if dryRun {
		log.Printf("Previewing monthly bills for %d-%02d", currentYear, currentMonth)
	} else {
		log.Printf("Generating monthly bills for %d-%02d", currentYear, currentMonth)
	}

	for _, sub := range subs {
		item := BillGenerationItem{
			SubscriptionID: sub.ID,
			CustomerID:     sub.CustomerID,
			CustomerName:   subscriptionCustomerName(&sub),
			PackageName:    subscriptionPackageName(&sub),
		}

		if !shouldGenerateBillForMonth(sub, billMonthStart, loc) {
			result.SkippedOutOfPeriod++
			item.Action = "skip"
			item.Reason = "subscription outside billing period"
			result.Items = append(result.Items, item)
			continue
		}
		result.Eligible++

		if sub.Package == nil {
			result.SkippedInvalid++
			item.Action = "skip"
			item.Reason = "subscription package is missing"
			result.Items = append(result.Items, item)
			continue
		}

		// Cek apakah sudah ada bill periode ini. Fallback ke bill_date disediakan
		// untuk data lama yang belum/backfill period field.
		existing, err := s.repo.FindBillBySubscriptionAndPeriod(sub.ID, currentYear, currentMonth)

		if err == nil && existing != nil {
			result.SkippedExisting++
			item.Action = "skip"
			item.Reason = "bill already exists for period"
			item.ExistingBillID = existing.ID.String()
			item.PublicID = existing.PublicID
			result.Items = append(result.Items, item)
			continue
		}
		if err != nil && err != gorm.ErrRecordNotFound {
			return nil, err
		}

		billDate := now
		dueDate := time.Date(currentYear, time.Month(currentMonth), sub.DueDay, 23, 59, 59, 0, loc)
		if dueDate.Month() != time.Month(currentMonth) {
			dueDate = time.Date(currentYear, time.Month(currentMonth)+1, 1, 23, 59, 59, 0, loc).AddDate(0, 0, -1)
		}

		amount := sub.Package.Price
		ppn := 0
		if sub.IsIncludePPN {
			ppn = int(float64(sub.Package.Price) * 0.11)
			amount += ppn // tambahkan 11% PPN
		}

		// Generate unique code (1–500) menggunakan crypto/rand
		uniqueCode := 0
		if sub.IsActiveUniqueCode {
			n, err := rand.Int(rand.Reader, big.NewInt(500))
			if err == nil {
				uniqueCode = int(n.Int64()) + 1 // hasil 1–500
			}
			amount += uniqueCode // tambahkan ke total
		}

		periodYear := currentYear
		periodMonth := currentMonth
		periodStart := billMonthStart
		periodEndCopy := periodEnd

		result.WouldCreate++
		result.EstimatedAmount += amount
		item.Action = "create"
		item.Reason = "eligible"
		item.Amount = amount

		if dryRun {
			result.Items = append(result.Items, item)
			continue
		}

		bill := &models.Bill{
			ID:             uuid.New(),
			PublicID:       fmt.Sprintf("%d%02d-%s", currentYear, currentMonth, uuid.NewString()[:6]),
			SubscriptionID: sub.ID,
			CustomerID:     sub.CustomerID,
			BillDate:       billDate,
			DueDate:        dueDate,
			Amount:         amount,
			PPN:            ppn,
			UniqueCode:     uniqueCode,
			Status:         "unpaid",
			PeriodYear:     &periodYear,
			PeriodMonth:    &periodMonth,
			PeriodStart:    &periodStart,
			PeriodEnd:      &periodEndCopy,
			Source:         "manual_generate",
			StatusReason:   "generated monthly bill",
			CreatedAt:      now,
			UpdatedAt:      now,
		}
		applyBillSnapshot(bill, &sub)

		log.Printf("Creating bill %s for customer %s: %v", bill.PublicID, sub.CustomerID.String(), bill)

		if err := s.repo.Create(bill); err != nil {
			return nil, fmt.Errorf("failed to create bill: %w", err)
		}
		result.Created++
		item.PublicID = bill.PublicID
		result.Items = append(result.Items, item)
	}

	// go func() {
	// 	log.Println("[WA] Sending reminders in background...")
	// 	if _, err := s.SendReminders(); err != nil {
	// 		log.Printf("[WA] Failed sending reminders: %v", err)
	// 	}
	// }()

	return result, nil
}

func (s *billService) MarkOverdueBills(referenceTime time.Time, limit int) (int, error) {
	if referenceTime.IsZero() {
		referenceTime = time.Now()
	}
	bills, err := s.repo.FindUnpaidOverdueBills(referenceTime, limit)
	if err != nil {
		return 0, err
	}
	updated := 0
	for i := range bills {
		bill := bills[i]
		now := time.Now()
		bill.Status = "overdue"
		if bill.OverdueAt == nil {
			bill.OverdueAt = &now
		}
		bill.StatusReason = "billing automation marked overdue"
		bill.UpdatedAt = now
		if err := s.repo.Update(&bill); err != nil {
			return updated, err
		}
		updated++
		if s.billingProvisioningSvc == nil {
			continue
		}
		subscription, subErr := s.subRepo.FindByID(bill.SubscriptionID)
		if subErr != nil {
			log.Printf("[billing-automation] failed to load subscription %s for overdue bill %s: %v", bill.SubscriptionID, bill.ID, subErr)
			continue
		}
		s.billingProvisioningSvc.HandleBillOverdue(&bill, subscription)
	}
	return updated, nil
}

func resolveBillingPeriod(period string, referenceTime time.Time, loc *time.Location) (time.Time, int, int, error) {
	if loc == nil {
		loc = time.Local
	}
	period = strings.TrimSpace(period)
	if period == "" {
		localRef := referenceTime.In(loc)
		start := time.Date(localRef.Year(), localRef.Month(), 1, 0, 0, 0, 0, loc)
		return start, localRef.Year(), int(localRef.Month()), nil
	}
	parsed, err := time.ParseInLocation("2006-01", period, loc)
	if err != nil {
		return time.Time{}, 0, 0, errors.New("invalid period format, expected YYYY-MM")
	}
	start := time.Date(parsed.Year(), parsed.Month(), 1, 0, 0, 0, 0, loc)
	return start, parsed.Year(), int(parsed.Month()), nil
}

func ensureBillPeriod(bill *models.Bill, loc *time.Location, source string) {
	if bill == nil {
		return
	}
	if loc == nil {
		loc = time.Local
	}
	baseDate := bill.BillDate
	if baseDate.IsZero() {
		baseDate = time.Now().In(loc)
	}
	localDate := baseDate.In(loc)
	year := localDate.Year()
	month := int(localDate.Month())
	start := time.Date(year, localDate.Month(), 1, 0, 0, 0, 0, loc)
	end := start.AddDate(0, 1, 0).Add(-time.Second)
	if bill.PeriodYear == nil {
		bill.PeriodYear = &year
	}
	if bill.PeriodMonth == nil {
		bill.PeriodMonth = &month
	}
	if bill.PeriodStart == nil {
		bill.PeriodStart = &start
	}
	if bill.PeriodEnd == nil {
		bill.PeriodEnd = &end
	}
	if strings.TrimSpace(bill.Source) == "" {
		bill.Source = source
	}
}

func subscriptionCustomerName(sub *models.Subscription) string {
	if sub == nil || sub.Customer == nil || sub.Customer.User == nil {
		return ""
	}
	return sub.Customer.User.Name
}

func subscriptionPackageName(sub *models.Subscription) string {
	if sub == nil || sub.Package == nil {
		return ""
	}
	return sub.Package.Name
}

func (s *billService) StartOverdueScheduler() {
	if !billingOverdueAutomationEnabled() {
		return
	}
	interval := resolveBillingOverdueInterval()
	go func() {
		if updated, err := s.MarkOverdueBills(time.Now(), 500); err != nil {
			log.Printf("[billing-automation] initial overdue run failed: %v", err)
		} else if updated > 0 {
			log.Printf("[billing-automation] initial overdue run marked %d bills overdue", updated)
		}
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			updated, err := s.MarkOverdueBills(time.Now(), 500)
			if err != nil {
				log.Printf("[billing-automation] scheduled overdue run failed: %v", err)
				continue
			}
			if updated > 0 {
				log.Printf("[billing-automation] scheduled overdue run marked %d bills overdue", updated)
			}
		}
	}()
}

func shouldGenerateBillForMonth(sub models.Subscription, billMonthStart time.Time, loc *time.Location) bool {
	billMonthStart = monthStartInLocation(billMonthStart, loc)

	if !sub.StartDate.IsZero() {
		subStartMonth := monthStartInLocation(sub.StartDate, loc)
		if billMonthStart.Before(subStartMonth) {
			return false
		}
	}

	if !sub.EndDate.IsZero() {
		subEndMonth := monthStartInLocation(sub.EndDate, loc)
		if subEndMonth.Before(billMonthStart) {
			return false
		}
	}

	return true
}

func monthStartInLocation(date time.Time, loc *time.Location) time.Time {
	if loc == nil {
		loc = time.Local
	}
	localDate := date.In(loc)
	return time.Date(localDate.Year(), localDate.Month(), 1, 0, 0, 0, 0, loc)
}

func billMonthRange(year int, month int) (time.Time, time.Time) {
	startOfMonth := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local)
	return startOfMonth, startOfMonth.AddDate(0, 1, 0)
}

func billingOverdueAutomationEnabled() bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv("BILLING_AUTOMATION_ENABLED")))
	return value == "1" || value == "true" || value == "yes"
}

func resolveBillingOverdueInterval() time.Duration {
	value := strings.TrimSpace(os.Getenv("BILLING_OVERDUE_CHECK_INTERVAL"))
	if value == "" {
		return time.Hour
	}
	interval, err := time.ParseDuration(value)
	if err != nil || interval <= 0 {
		return time.Hour
	}
	return interval
}

func (s *billService) GetUnpaidBills() ([]models.Bill, error) {
	return s.repo.FindUnpaidBills()
}

func (s *billService) SendReminders() (map[string]interface{}, error) {
	bills, err := s.repo.FindUnpaidBills()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch unpaid bills: %w", err)
	}

	stats := newReminderStats(len(bills))

	baseTime := time.Now()
	delayIndex := 1

	for _, bill := range bills {
		phone, ok := s.getValidPhone(bill)
		if !ok {
			stats.skipped++
			continue
		}

		billMessage := utils.BuildBillMessage(bill)
		reminderMessage := utils.BuildReminderMessage(bill)
		billSendTime := baseTime.Add(time.Duration(delayIndex*3) * time.Second)
		reminderSendTime := bill.DueDate.Add(time.Duration(delayIndex*3) * time.Second)

		if err := s.waSvc.SendScheduledMessage(phone, billMessage, billSendTime); err != nil {
			stats.failed++
			stats.addError(phone, bill.PublicID, err)
			continue
		}

		if err := s.waSvc.SendScheduledMessage(phone, reminderMessage, reminderSendTime); err != nil {
			stats.failed++
			stats.addError(phone, bill.PublicID, err)
			continue
		}

		stats.sent++
		delayIndex++
		log.Printf("Sent reminder for bill %s to %s", bill.PublicID, phone)
	}

	return stats.toMap(), nil
}

func (s *billService) getValidPhone(bill models.Bill) (string, bool) {
	if bill.Customer.User == nil || !bill.Customer.IsSendWa {
		return "", false
	}

	rawPhone := bill.Customer.User.Phone
	if rawPhone == "" {
		log.Printf("Skipping bill %s: customer has no phone number", bill.PublicID)
		return "", false
	}

	phone := utils.NormalizeIDPhone(rawPhone)
	if phone == "" {
		log.Printf("Skipping bill %s: invalid phone number %q", bill.PublicID, rawPhone)
		return "", false
	}

	return phone, true
}

type reminderStats struct {
	total   int
	sent    int
	skipped int
	failed  int
	errors  []string
}

type billDashboardTrend struct {
	Month         string `json:"month"`
	MonthKey      string `json:"month_key"`
	Paid          int64  `json:"paid"`
	Unpaid        int64  `json:"unpaid"`
	Overdue       int64  `json:"overdue"`
	Total         int64  `json:"total"`
	AmountPaid    int64  `json:"amount_paid"`
	AmountUnpaid  int64  `json:"amount_unpaid"`
	AmountOverdue int64  `json:"amount_overdue"`
	AmountTotal   int64  `json:"amount_total"`
}

func newReminderStats(total int) *reminderStats {
	return &reminderStats{total: total}
}

func (r *reminderStats) addError(phone, billID string, err error) {
	r.errors = append(r.errors,
		fmt.Sprintf("Failed to send to %s (bill %s): %v", phone, billID, err),
	)
}

func (r *reminderStats) toMap() map[string]interface{} {
	result := map[string]interface{}{
		"total_unpaid_bills": r.total,
		"messages_sent":      r.sent,
		"messages_skipped":   r.skipped,
		"messages_failed":    r.failed,
	}

	if len(r.errors) > 0 {
		result["errors"] = r.errors
	}

	return result
}

func (s *billService) GetDashboardStats(month, year int, adminID string) (map[string]interface{}, error) {
	adminID = strings.TrimSpace(adminID)
	var parsedAdminID *uuid.UUID
	if adminID != "" {
		uid, err := uuid.Parse(adminID)
		if err == nil {
			parsedAdminID = &uid
		}
	}

	stats, err := s.repo.GetDashboardStats(month, year, parsedAdminID)
	if err != nil {
		return nil, err
	}

	// Convert int64 values to interface{} map
	result := make(map[string]interface{})
	for key, value := range stats {
		result[key] = value
	}

	return result, nil
}

func (s *billService) GetRecentPaidBills(limit int) ([]models.Bill, error) {
	return s.repo.GetRecentPaidBills(limit)
}

func (s *billService) GetDashboardCharts(months int, adminID string) (map[string]interface{}, error) {
	if months <= 0 {
		months = 6
	}
	if months > 24 {
		months = 24
	}

	adminID = strings.TrimSpace(adminID)
	var parsedAdminID *uuid.UUID
	if adminID != "" {
		uid, err := uuid.Parse(adminID)
		if err != nil {
			return nil, errors.New("invalid admin_id")
		}
		parsedAdminID = &uid
	}

	now := time.Now()
	startDate := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local).AddDate(0, -(months - 1), 0)

	bills, err := s.repo.GetDashboardChartRows(startDate, parsedAdminID)
	if err != nil {
		return nil, err
	}

	trend := make([]billDashboardTrend, 0, months)
	monthIndex := make(map[string]int, months)

	for i := 0; i < months; i++ {
		monthDate := startDate.AddDate(0, i, 0)
		key := monthDate.Format("2006-01")
		monthIndex[key] = i
		trend = append(trend, billDashboardTrend{
			Month:    monthDate.Format("Jan 2006"),
			MonthKey: key,
		})
	}

	statusTotals := map[string]int64{
		"paid":    0,
		"unpaid":  0,
		"overdue": 0,
	}
	amountTotals := map[string]int64{
		"paid":    0,
		"unpaid":  0,
		"overdue": 0,
	}

	for _, bill := range bills {
		key := bill.BillDate.Format("2006-01")
		idx, exists := monthIndex[key]
		if !exists {
			continue
		}

		status := strings.ToLower(strings.TrimSpace(bill.Status))
		isOverdue := status == "overdue" || (status == "unpaid" && bill.DueDate.Before(now))
		if status == "paid" {
			trend[idx].Paid++
			trend[idx].AmountPaid += int64(bill.Amount)
			statusTotals["paid"]++
			amountTotals["paid"] += int64(bill.Amount)
		} else if isOverdue {
			trend[idx].Overdue++
			trend[idx].AmountOverdue += int64(bill.Amount)
			statusTotals["overdue"]++
			amountTotals["overdue"] += int64(bill.Amount)
		} else {
			trend[idx].Unpaid++
			trend[idx].AmountUnpaid += int64(bill.Amount)
			statusTotals["unpaid"]++
			amountTotals["unpaid"] += int64(bill.Amount)
		}

		trend[idx].Total++
		trend[idx].AmountTotal += int64(bill.Amount)
	}

	return map[string]interface{}{
		"months":        months,
		"range_start":   startDate,
		"range_end":     now,
		"trend":         trend,
		"status_totals": statusTotals,
		"amount_totals": amountTotals,
	}, nil
}

func parseBillDateRange(startAt string, endAt string) (*time.Time, *time.Time, error) {
	var startDate *time.Time
	var endDate *time.Time
	var rawStart *time.Time
	var rawEnd *time.Time

	if startAt != "" {
		parsedStart, err := time.Parse("2006-01-02", startAt)
		if err != nil {
			return nil, nil, errors.New("invalid start_at format, expected YYYY-MM-DD")
		}
		rawStart = &parsedStart
		start := time.Date(parsedStart.Year(), parsedStart.Month(), parsedStart.Day(), 0, 0, 0, 0, time.UTC)
		startDate = &start
	}

	if endAt != "" {
		parsedEnd, err := time.Parse("2006-01-02", endAt)
		if err != nil {
			return nil, nil, errors.New("invalid end_at format, expected YYYY-MM-DD")
		}
		rawEnd = &parsedEnd
		end := time.Date(parsedEnd.Year(), parsedEnd.Month(), parsedEnd.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, 1)
		endDate = &end
	}

	if rawStart != nil && rawEnd != nil && rawStart.After(*rawEnd) {
		return nil, nil, errors.New("start_at must be before or equal end_at")
	}

	return startDate, endDate, nil
}
