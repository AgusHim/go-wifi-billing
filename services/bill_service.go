package services

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/Agushim/go_wifi_billing/models"
	"github.com/Agushim/go_wifi_billing/repositories"
	"github.com/Agushim/go_wifi_billing/utils"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BillService interface {
	GetAll(page, limit int, search string) ([]models.Bill, int64, error)
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
	GetRecentPaidBills(limit int) ([]models.Bill, error)
}

type billService struct {
	repo    repositories.BillRepository
	subRepo repositories.SubscriptionRepository
	waSvc   WhatsAppService
}

func NewBillService(repo repositories.BillRepository, subRepo repositories.SubscriptionRepository, waSvc WhatsAppService) BillService {
	return &billService{repo, subRepo, waSvc}
}

func (s *billService) GetAll(page, limit int, search string) ([]models.Bill, int64, error) {
	return s.repo.FindAllPaginated(page, limit, search)
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
	bill.Amount = input.Amount
	bill.Status = input.Status
	bill.BillDate = input.BillDate
	bill.DueDate = input.DueDate
	bill.TerminatedDate = input.TerminatedDate
	bill.UpdatedAt = time.Now()
	err = s.repo.Update(&bill)
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
		existing, err := s.repo.FindBillByCustomerAndMonth(sub.CustomerID.String(), currentMonth, currentYear)

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
		ppn := int(float64(sub.Package.Price) * 0.11)
		if sub.IsIncludePPN {
			amount = int(float64(amount) * 1.11) // tambahkan 11% PPN
		}

		// ✅ Generate unique code (001–500)
		uniqueCode := 0
		if sub.IsActiveUniqueCode {
			uniqueCode = rand.Intn(799) + 1 // hasil 1–500
			amount += uniqueCode            // tambahkan ke total
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
