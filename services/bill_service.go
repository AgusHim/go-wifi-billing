package services

import (
	"crypto/rand"
	"errors"
	"fmt"
	"log"
	"math/big"
	"strings"
	"time"

	"github.com/Agushim/go_wifi_billing/models"
	"github.com/Agushim/go_wifi_billing/repositories"
	"github.com/Agushim/go_wifi_billing/utils"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BillService interface {
	GetAll(page, limit int, search string, adminID string, status string, startAt string, endAt string, coverageID string) ([]models.Bill, int64, error)
	GetByID(id string) (models.Bill, error)
	Create(input models.Bill) (models.Bill, error)
	Update(id string, input models.Bill) (models.Bill, error)
	Delete(id string) error
	GenerateMonthlyBills() error
	GetByPublicID(publicID string) (*models.Bill, error)
	GetByUserID(userID string) ([]models.Bill, error)
	GetUnpaidBills() ([]models.Bill, error)
	SendReminders() (map[string]interface{}, error)
	GetDashboardStats() (map[string]interface{}, error)
	GetDashboardCharts(months int, adminID string) (map[string]interface{}, error)
	GetRecentPaidBills(limit int) ([]models.Bill, error)
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

func (s *billService) GetAll(page, limit int, search string, adminID string, status string, startAt string, endAt string, coverageID string) ([]models.Bill, int64, error) {
	adminID = strings.TrimSpace(adminID)
	status = strings.TrimSpace(strings.ToLower(status))
	startAt = strings.TrimSpace(startAt)
	endAt = strings.TrimSpace(endAt)
	coverageID = strings.TrimSpace(coverageID)

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

	var parsedCoverageID *uuid.UUID
	if coverageID != "" {
		uid, err := uuid.Parse(coverageID)
		if err != nil {
			return nil, 0, errors.New("invalid coverage_id")
		}
		parsedCoverageID = &uid
	}

	return s.repo.FindAllPaginated(page, limit, search, parsedAdminID, status, startDate, endDate, parsedCoverageID)
}

func (s *billService) GetByID(id string) (models.Bill, error) {
	return s.repo.FindByID(id)
}

func (s *billService) Create(input models.Bill) (models.Bill, error) {
	input.ID = uuid.New()
	input.CreatedAt = time.Now()
	input.UpdatedAt = time.Now()
	err := s.repo.Create(&input)
	return input, err
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

func (s *billService) GenerateMonthlyBills() error {
	status := "active"
	subs, err := s.subRepo.FindForBill(nil, &status, true)
	log.Printf("Found %d active subscriptions", len(subs))
	if err != nil {
		return fmt.Errorf("failed to fetch subscriptions: %w", err)
	}

	currentMonth := int(time.Now().Month())
	currentYear := time.Now().Year()

	log.Printf("Generating monthly bills for %d-%02d", currentYear, currentMonth)

	for _, sub := range subs {
		// Cek apakah sudah ada bill bulan ini
		existing, err := s.repo.FindBillBySubscriptionAndMonth(sub.ID, currentMonth, currentYear)

		if err == nil && existing != nil {
			continue // sudah ada bill bulan ini
		}
		if err != nil && err != gorm.ErrRecordNotFound {
			return err
		}

		billDate := time.Now()
		dueDate := time.Date(currentYear, time.Month(currentMonth), sub.DueDay, 23, 59, 59, 0, time.Local)
		if dueDate.Month() != time.Month(currentMonth) {
			dueDate = time.Date(currentYear, time.Month(currentMonth)+1, 1, 23, 59, 59, 0, time.Local).AddDate(0, 0, -1)
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
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		log.Printf("Creating bill %s for customer %s: %v", bill.PublicID, sub.CustomerID.String(), bill)

		if err := s.repo.Create(bill); err != nil {
			return fmt.Errorf("failed to create bill: %w", err)
		}
	}

	// go func() {
	// 	log.Println("[WA] Sending reminders in background...")
	// 	if _, err := s.SendReminders(); err != nil {
	// 		log.Printf("[WA] Failed sending reminders: %v", err)
	// 	}
	// }()

	return nil
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

func (s *billService) GetDashboardStats() (map[string]interface{}, error) {
	stats, err := s.repo.GetDashboardStats()
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
		switch status {
		case "paid":
			trend[idx].Paid++
			trend[idx].AmountPaid += int64(bill.Amount)
			statusTotals["paid"]++
			amountTotals["paid"] += int64(bill.Amount)
		case "overdue":
			trend[idx].Overdue++
			trend[idx].AmountOverdue += int64(bill.Amount)
			statusTotals["overdue"]++
			amountTotals["overdue"] += int64(bill.Amount)
		default:
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
