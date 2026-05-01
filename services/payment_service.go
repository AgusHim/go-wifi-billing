package services

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/Agushim/go_wifi_billing/models"
	"github.com/Agushim/go_wifi_billing/repositories"
	"github.com/google/uuid"
	"github.com/midtrans/midtrans-go"
	"github.com/midtrans/midtrans-go/snap"
	"gorm.io/gorm"
)

type PaymentService interface {
	GetAll(adminID string, search string, status string, startAt string, endAt string, page int, limit int) ([]models.Payment, int64, error)
	GetByID(id string) (models.Payment, error)
	Create(input models.Payment) (*models.Payment, error)
	Update(id string, input models.Payment) (*models.Payment, error)
	Delete(id string) error
	CreateMidtransTransaction(paymentID string) (*models.Payment, error)
	HandleMindtransCallback(paymentID string, status string) error
	GetByUserID(userID string) ([]models.Payment, error)
	BatchCreate(inputs []models.Payment) ([]models.Payment, error)
	ExportCSV(adminID string, search string, status string, startAt string, endAt string) ([]byte, error)
}

type paymentService struct {
	repo                   repositories.PaymentRepository
	subcRepo               repositories.SubscriptionRepository
	billRepo               repositories.BillRepository
	billingProvisioningSvc BillingProvisioningService
	renewalSvc             RenewalService
}

func NewPaymentService(
	repo repositories.PaymentRepository,
	subcRepo repositories.SubscriptionRepository,
	billRepo repositories.BillRepository,
	billingProvisioningSvc BillingProvisioningService,
	renewalSvc RenewalService,
) PaymentService {
	env := os.Getenv("MIDTRANS_ENV")
	if env == "sandbox" {
		midtrans.Environment = midtrans.Sandbox
	} else {
		midtrans.Environment = midtrans.Production
	}
	midtrans.ServerKey = os.Getenv("MIDTRANS_SERVER_KEY")
	midtrans.ClientKey = os.Getenv("MIDTRANS_CLIENT_KEY")
	return &paymentService{
		repo:                   repo,
		subcRepo:               subcRepo,
		billRepo:               billRepo,
		billingProvisioningSvc: billingProvisioningSvc,
		renewalSvc:             renewalSvc,
	}
}

func (s *paymentService) GetAll(adminID string, search string, status string, startAt string, endAt string, page int, limit int) ([]models.Payment, int64, error) {
	adminID = strings.TrimSpace(adminID)
	search = strings.TrimSpace(search)
	status = strings.TrimSpace(status)
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

	startDate, endDate, err := parseStartEndRange(startAt, endAt)
	if err != nil {
		return nil, 0, err
	}

	return s.repo.FindAll(parsedAdminID, search, status, startDate, endDate, page, limit)
}

func (s *paymentService) GetByID(id string) (models.Payment, error) {
	return s.repo.FindByID(id)
}
func (s *paymentService) GetByUserID(userID string) ([]models.Payment, error) {
	return s.repo.FindByUserID(userID)
}

func (s *paymentService) Create(input models.Payment) (*models.Payment, error) {
	bill, err := s.billRepo.FindByID(input.BillID.String())
	if err != nil {
		return nil, err
	}

	// Cegah double payment: cek apakah sudah ada payment confirmed/pending untuk bill ini
	existing, _ := s.repo.FindActiveByBillID(input.BillID)
	if existing != nil {
		return nil, fmt.Errorf("bill sudah memiliki payment aktif (status: %s)", existing.Status)
	}

	input.ID = uuid.New()
	input.CreatedAt = time.Now()
	input.UpdatedAt = time.Now()
	err = s.repo.Create(&input)
	if err != nil {
		return nil, err
	}

	if strings.EqualFold(strings.TrimSpace(input.Status), "confirmed") {
		nbill, nsubs, nerr := s.handleConfirmation(&input, &bill)
		if nerr != nil {
			return nil, nerr
		}
		input.Bill = *nbill
		input.Bill.Subscription = *nsubs
	}
	return &input, nil
}

func (s *paymentService) BatchCreate(inputs []models.Payment) ([]models.Payment, error) {
	var results []models.Payment
	for _, input := range inputs {
		res, err := s.Create(input)
		if err != nil {
			return nil, err
		}
		results = append(results, *res)
	}
	return results, nil
}

func (s *paymentService) Update(id string, input models.Payment) (*models.Payment, error) {
	payment, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	previousStatus := strings.TrimSpace(strings.ToLower(payment.Status))

	bill, err := s.billRepo.FindByID(input.BillID.String())
	if err != nil {
		return nil, err
	}

	payment.RefID = input.RefID
	payment.PaymentDate = input.PaymentDate
	payment.DueDate = input.DueDate
	payment.Method = input.Method
	payment.Amount = input.Amount
	payment.Status = input.Status
	payment.UpdatedAt = time.Now()

	err = s.repo.Update(&payment)
	if err != nil {
		return nil, err
	}

	// Update Bill And Subscription
	if previousStatus != "confirmed" && strings.EqualFold(strings.TrimSpace(payment.Status), "confirmed") {
		nbill, nsubs, nerr := s.handleConfirmation(&payment, &bill)
		if nerr != nil {
			return nil, nerr
		}
		payment.Bill = *nbill
		payment.Bill.Subscription = *nsubs
	}

	return &payment, nil
}

func (s *paymentService) Delete(id string) error {
	payment, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}

	// Rollback bill and subscribe
	if payment.Status == "confirmed" {
		_, _, err := s.RollbackBillAndSubs(payment.Bill)
		if err != nil {
			return err
		}
	}
	return s.repo.Delete(id)
}

func (s *paymentService) UpdateBillAndSubs(input models.Payment, bill models.Bill) (*models.Bill, *models.Subscription, error) {

	// Update bill status to paid
	bill.Status = "paid"
	err := s.billRepo.Update(&bill)
	if err != nil {
		return nil, nil, err
	}

	// Update subscription duration
	subs, err := s.subcRepo.FindByID(bill.SubscriptionID)
	if err != nil {
		return nil, nil, err
	}
	endSubs := subs.EndDate
	subs.StartDate = endSubs
	subs.EndDate = endSubs.AddDate(0, 1, 0)

	err = s.subcRepo.Update(subs)
	if err != nil {
		return nil, nil, err
	}
	return &bill, subs, nil
}

func (s *paymentService) handleConfirmation(payment *models.Payment, bill *models.Bill) (*models.Bill, *models.Subscription, error) {
	if strings.EqualFold(strings.TrimSpace(bill.Status), "paid") {
		subscription, err := s.subcRepo.FindByID(bill.SubscriptionID)
		if err != nil {
			return nil, nil, err
		}
		return bill, subscription, nil
	}

	nbill, nsubs, err := s.UpdateBillAndSubs(*payment, *bill)
	if err != nil {
		return nil, nil, err
	}

	if s.billingProvisioningSvc != nil {
		s.billingProvisioningSvc.HandlePaymentConfirmed(payment, nbill, nsubs)
	}
	if s.renewalSvc != nil {
		_ = s.renewalSvc.RecordPaymentConfirmed(nsubs.ID, nbill.ID, payment.ID, fmt.Sprintf("Payment %s confirmed", payment.ID))
	}

	return nbill, nsubs, nil
}

func (s *paymentService) RollbackBillAndSubs(bill models.Bill) (*models.Bill, *models.Subscription, error) {
	// Rollback bill status to unpaid
	bill.Status = "unpaid"
	err := s.billRepo.Update(&bill)
	if err != nil {
		return nil, nil, err
	}

	// Update subscription duration
	subs, err := s.subcRepo.FindByID(bill.SubscriptionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Subscription was deleted/terminated, skip date rollback
			return &bill, nil, nil
		}
		return nil, nil, err
	}

	// Mundurkan 1 bulan masa berlangganan
	subs.EndDate = subs.EndDate.AddDate(0, -1, 0)
	subs.StartDate = subs.StartDate.AddDate(0, -1, 0)

	err = s.subcRepo.Update(subs)
	if err != nil {
		return nil, nil, err
	}

	return &bill, subs, nil
}

func (s *paymentService) CreateMidtransTransaction(billID string) (*models.Payment, error) {
	bill, err := s.billRepo.FindByID(billID)
	if err != nil {
		return nil, err
	}

	// Cegah duplikat: cek apakah sudah ada payment confirmed/pending untuk bill ini
	existing, _ := s.repo.FindActiveByBillID(bill.ID)
	if existing != nil {
		return nil, fmt.Errorf("bill sudah memiliki payment aktif (status: %s)", existing.Status)
	}

	now := time.Now()
	var payment models.Payment
	payment.Bill = bill
	payment.ID = uuid.New()
	payment.BillID = bill.ID
	payment.PaymentDate = now
	payment.DueDate = bill.DueDate
	payment.ExpiredDate = now.AddDate(0, 0, 1)
	payment.Method = "midtrans"
	payment.Amount = bill.Amount
	payment.Status = "pending"
	payment.AdminID = nil
	payment.CreatedAt = now
	payment.UpdatedAt = now

	// Create midtrans payment
	req := &snap.Request{
		TransactionDetails: midtrans.TransactionDetails{
			OrderID:  payment.ID.String(),
			GrossAmt: int64(payment.Amount),
		},
	}
	if bill.Customer.User != nil {
		req.CustomerDetail = &midtrans.CustomerDetails{
			FName: bill.Customer.User.Name,
			Email: bill.Customer.User.Email,
		}
	}

	snapResp, nerr := snap.CreateTransaction(req)
	if nerr != nil {
		log.Println("Midtrans error:", nerr)
		return nil, nerr
	}
	payment.RefID = snapResp.Token
	payment.PaymentUrl = &snapResp.RedirectURL

	err = s.repo.Create(&payment)
	if err != nil {
		return nil, err
	}

	return &payment, nil
}

func (s *paymentService) HandleMindtransCallback(paymentID string, status string) error {
	payment, err := s.repo.FindByID(paymentID)
	if err != nil {
		return err
	}
	previousStatus := strings.TrimSpace(strings.ToLower(payment.Status))
	payment_status := getStatus(status)
	payment.Status = payment_status

	// Update Bill And Subscription
	if previousStatus != "confirmed" && payment.Status == "confirmed" {
		bill, nerr := s.billRepo.FindByID(payment.BillID.String())
		if nerr != nil {
			return nerr
		}
		nbill, nsubs, nerr := s.handleConfirmation(&payment, &bill)
		if nerr != nil {
			return nerr
		}
		payment.Bill = *nbill
		payment.Bill.Subscription = *nsubs
	}

	err = s.repo.Update(&payment)
	if err != nil {
		return err
	}
	return err
}

func getStatus(status string) string {
	if status == "settlement" || status == "capture" {
		return "confirmed"
	} else {
		return status
	}
}

func (s *paymentService) ExportCSV(adminID string, search string, status string, startAt string, endAt string) ([]byte, error) {
	payments, _, err := s.GetAll(adminID, search, status, startAt, endAt, 0, 0)
	if err != nil {
		return nil, err
	}

	buffer := &bytes.Buffer{}
	writer := csv.NewWriter(buffer)

	headers := []string{
		"invoice_number",
		"customer_name",
		"customer_email",
		"method",
		"payment_date",
		"amount",
		"status",
		"admin_name",
		"created_at",
		"updated_at",
	}
	if err := writer.Write(headers); err != nil {
		return nil, err
	}

	for _, payment := range payments {
		invoice := ""
		customerName := ""
		customerEmail := ""
		adminName := ""

		if payment.Bill.PublicID != "" {
			invoice = strings.ToUpper(payment.Bill.PublicID)
		}
		if payment.Bill.Customer.User != nil {
			customerName = payment.Bill.Customer.User.Name
			customerEmail = payment.Bill.Customer.User.Email
		}
		if payment.Admin.Name != "" {
			adminName = payment.Admin.Name
		}

		record := []string{
			invoice,
			customerName,
			customerEmail,
			payment.Method,
			payment.PaymentDate.Format(time.RFC3339),
			fmt.Sprintf("%d", payment.Amount),
			payment.Status,
			adminName,
			payment.CreatedAt.Format(time.RFC3339),
			payment.UpdatedAt.Format(time.RFC3339),
		}

		if err := writer.Write(record); err != nil {
			return nil, err
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func parseStartEndRange(startAt string, endAt string) (*time.Time, *time.Time, error) {
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
