package services

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/Agushim/go_wifi_billing/lib"
	"github.com/Agushim/go_wifi_billing/models"
	"github.com/Agushim/go_wifi_billing/repositories"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type VoucherService interface {
	CreateBatch(input *models.VoucherBatch) (*models.VoucherBatch, error)
	GetBatches() ([]models.VoucherBatch, error)
	GetBatchByID(id uuid.UUID) (*models.VoucherBatch, error)
	GetVouchers(batchID string) ([]models.Voucher, error)
	GetVoucherByID(id uuid.UUID) (*models.Voucher, error)
	Redeem(code string, redeemerName string, redeemerPhone string) (*models.Voucher, *models.ProvisioningJob, error)
}

type voucherService struct {
	batchRepo   repositories.VoucherBatchRepository
	voucherRepo repositories.VoucherRepository
	jobRepo     repositories.ProvisioningJobRepository
	logRepo     repositories.ProvisioningLogRepository
	routerRepo  repositories.RouterRepository
}

func NewVoucherService(
	batchRepo repositories.VoucherBatchRepository,
	voucherRepo repositories.VoucherRepository,
	jobRepo repositories.ProvisioningJobRepository,
	logRepo repositories.ProvisioningLogRepository,
	routerRepo repositories.RouterRepository,
) VoucherService {
	return &voucherService{
		batchRepo:   batchRepo,
		voucherRepo: voucherRepo,
		jobRepo:     jobRepo,
		logRepo:     logRepo,
		routerRepo:  routerRepo,
	}
}

func (s *voucherService) CreateBatch(input *models.VoucherBatch) (*models.VoucherBatch, error) {
	if err := validateVoucherBatch(input); err != nil {
		return nil, err
	}

	if input.NetworkPlanID != nil && input.NetworkPlan == nil {
		// no-op, preload will happen on re-fetch
	}
	if strings.TrimSpace(input.Status) == "" {
		input.Status = "active"
	}
	if err := s.batchRepo.Create(input); err != nil {
		return nil, err
	}

	for i := 0; i < input.Quantity; i++ {
		password := randomToken(8)
		encrypted, err := lib.EncryptSecret(password)
		if err != nil {
			return nil, err
		}

		voucher := &models.Voucher{
			BatchID:           input.ID,
			Code:              fmt.Sprintf("%s-%s", sanitizeVoucherPrefix(input.Prefix), randomToken(6)),
			Username:          buildVoucherUsername(input.Prefix, i),
			PasswordEncrypted: encrypted,
			ServiceType:       normalizeVoucherServiceType(input.ServiceType),
			RouterID:          input.RouterID,
			NetworkPlanID:     input.NetworkPlanID,
			Status:            "generated",
		}
		if err := s.voucherRepo.Create(voucher); err != nil {
			return nil, err
		}
	}

	return s.GetBatchByID(input.ID)
}

func (s *voucherService) GetBatches() ([]models.VoucherBatch, error) {
	items, err := s.batchRepo.FindAll()
	if err != nil {
		return nil, err
	}
	for i := range items {
		items[i] = *sanitizeVoucherBatch(&items[i])
	}
	return items, nil
}

func (s *voucherService) GetBatchByID(id uuid.UUID) (*models.VoucherBatch, error) {
	item, err := s.batchRepo.FindByID(id)
	if err != nil {
		return nil, err
	}
	return sanitizeVoucherBatch(item), nil
}

func (s *voucherService) GetVouchers(batchID string) ([]models.Voucher, error) {
	items, err := s.voucherRepo.FindAll(batchID)
	if err != nil {
		return nil, err
	}
	for i := range items {
		items[i] = *sanitizeVoucher(&items[i])
	}
	return items, nil
}

func (s *voucherService) GetVoucherByID(id uuid.UUID) (*models.Voucher, error) {
	item, err := s.voucherRepo.FindByID(id)
	if err != nil {
		return nil, err
	}
	return sanitizeVoucher(item), nil
}

func (s *voucherService) Redeem(code string, redeemerName string, redeemerPhone string) (*models.Voucher, *models.ProvisioningJob, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return nil, nil, errors.New("voucher code is required")
	}

	voucher, err := s.voucherRepo.FindByCode(code)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, errors.New("voucher not found")
		}
		return nil, nil, err
	}
	if voucher.Batch != nil && voucher.Batch.ExpiresAt != nil && voucher.Batch.ExpiresAt.Before(time.Now()) {
		return nil, nil, errors.New("voucher expired")
	}
	if strings.TrimSpace(strings.ToLower(voucher.Status)) != "generated" {
		return nil, nil, errors.New("voucher already redeemed")
	}

	now := time.Now()
	voucher.Status = "redeemed"
	voucher.RedeemedAt = &now
	voucher.RedeemerName = strings.TrimSpace(redeemerName)
	voucher.RedeemerPhone = strings.TrimSpace(redeemerPhone)
	if err := s.voucherRepo.Update(voucher); err != nil {
		return nil, nil, err
	}

	payloadMap := map[string]string{
		"voucher_id": voucher.ID.String(),
		"code":       voucher.Code,
		"username":   voucher.Username,
	}
	payloadBytes, _ := json.Marshal(payloadMap)
	job := &models.ProvisioningJob{
		EntityType: "voucher",
		EntityID:   &voucher.ID,
		Action:     "create_account",
		Payload:    string(payloadBytes),
		Status:     "pending",
	}
	if err := s.jobRepo.Create(job); err != nil {
		return nil, nil, err
	}
	s.log(voucher.RouterID, &job.ID, "info", "create_account", "voucher provisioning job queued", string(payloadBytes), "")

	if provisioningEnabled() {
		go s.processJob(job.ID)
	}

	return sanitizeVoucher(voucher), job, nil
}

func (s *voucherService) processJob(jobID uuid.UUID) {
	job, err := s.jobRepo.FindByID(jobID)
	if err != nil || job == nil {
		return
	}
	if job.EntityID == nil {
		return
	}

	voucher, err := s.voucherRepo.FindByID(*job.EntityID)
	if err != nil {
		job.Status = "failed"
		job.ErrorMessage = err.Error()
		now := time.Now()
		job.ExecutedAt = &now
		_ = s.jobRepo.Update(job)
		return
	}

	job.Status = "processing"
	job.AttemptCount++
	now := time.Now()
	job.ExecutedAt = &now
	_ = s.jobRepo.Update(job)

	if err := s.executeVoucherProvisioning(voucher, job); err != nil {
		job.Status = "failed"
		job.ErrorMessage = err.Error()
		_ = s.jobRepo.Update(job)
		s.log(voucher.RouterID, &job.ID, "error", job.Action, err.Error(), job.Payload, "")
		if job.AttemptCount < 3 && provisioningEnabled() {
			retryAt := time.Now().Add(time.Duration(job.AttemptCount*5) * time.Second)
			job.ScheduledAt = &retryAt
			_ = s.jobRepo.Update(job)
			go func(retryJobID uuid.UUID, delay time.Duration) {
				time.Sleep(delay)
				s.processJob(retryJobID)
			}(job.ID, time.Duration(job.AttemptCount*5)*time.Second)
		}
		return
	}

	job.Status = "success"
	job.ErrorMessage = ""
	_ = s.jobRepo.Update(job)
}

func (s *voucherService) executeVoucherProvisioning(voucher *models.Voucher, job *models.ProvisioningJob) error {
	router, err := resolveVoucherRouter(voucher)
	if err != nil {
		return err
	}

	routerRecord, err := s.routerRepo.FindByID(router.ID)
	if err != nil {
		return err
	}
	routerPassword, err := lib.DecryptSecret(routerRecord.PasswordEncrypted)
	if err != nil {
		return err
	}
	if strings.TrimSpace(routerPassword) == "" {
		return errors.New("router credential is not configured")
	}

	client, err := lib.NewMikrotikClient(routerRecord.Host, routerRecord.Port, routerRecord.UseTLS, 5*time.Second)
	if err != nil {
		return err
	}
	defer client.Close()

	if err := client.Login(routerRecord.Username, routerPassword); err != nil {
		return err
	}

	password, err := lib.DecryptSecret(voucher.PasswordEncrypted)
	if err != nil {
		return err
	}

	requestWords, err := voucherRequestWords(voucher, password)
	if err != nil {
		return err
	}

	response, err := client.Run(requestWords...)
	responseBytes, _ := json.Marshal(response)
	if err != nil {
		return err
	}

	now := time.Now()
	voucher.LastProvisionedAt = &now
	_ = s.voucherRepo.Update(voucher)
	s.log(voucher.RouterID, &job.ID, "info", "create_account", "voucher provisioning command succeeded", strings.Join(requestWords, " "), string(responseBytes))
	return nil
}

func voucherRequestWords(voucher *models.Voucher, password string) ([]string, error) {
	serviceType := normalizeVoucherServiceType(voucher.ServiceType)
	switch serviceType {
	case "pppoe":
		words := []string{
			"/ppp/secret/add",
			"=name=" + voucher.Username,
			"=password=" + password,
			"=service=pppoe",
		}
		if voucher.NetworkPlan != nil && strings.TrimSpace(voucher.NetworkPlan.MikrotikProfileName) != "" {
			words = append(words, "=profile="+voucher.NetworkPlan.MikrotikProfileName)
		}
		if voucher.NetworkPlan != nil && strings.TrimSpace(voucher.NetworkPlan.AddressPool) != "" {
			words = append(words, "=remote-address="+voucher.NetworkPlan.AddressPool)
		}
		return words, nil
	case "hotspot":
		words := []string{
			"/ip/hotspot/user/add",
			"=name=" + voucher.Username,
			"=password=" + password,
		}
		if voucher.NetworkPlan != nil && strings.TrimSpace(voucher.NetworkPlan.MikrotikProfileName) != "" {
			words = append(words, "=profile="+voucher.NetworkPlan.MikrotikProfileName)
		}
		return words, nil
	default:
		return nil, errors.New("unsupported voucher service type")
	}
}

func resolveVoucherRouter(voucher *models.Voucher) (*models.Router, error) {
	if voucher == nil {
		return nil, errors.New("voucher is nil")
	}
	if voucher.Router != nil {
		return voucher.Router, nil
	}
	if voucher.NetworkPlan != nil && voucher.NetworkPlan.Router != nil {
		return voucher.NetworkPlan.Router, nil
	}
	return nil, errors.New("voucher has no router mapping")
}

func sanitizeVoucher(input *models.Voucher) *models.Voucher {
	if input == nil {
		return nil
	}
	copy := *input
	password, err := lib.DecryptSecret(copy.PasswordEncrypted)
	if err == nil {
		copy.Password = password
	}
	return &copy
}

func validateVoucherBatch(input *models.VoucherBatch) error {
	if input == nil {
		return errors.New("invalid voucher batch payload")
	}
	if strings.TrimSpace(input.Name) == "" {
		return errors.New("name is required")
	}
	if strings.TrimSpace(input.Prefix) == "" {
		return errors.New("prefix is required")
	}
	if input.Quantity <= 0 {
		return errors.New("quantity must be greater than zero")
	}
	if input.Quantity > 500 {
		return errors.New("quantity must be less than or equal to 500")
	}
	input.ServiceType = normalizeVoucherServiceType(input.ServiceType)
	if input.ServiceType != "hotspot" && input.ServiceType != "pppoe" {
		return errors.New("service_type must be hotspot or pppoe")
	}
	if input.RouterID == nil && input.NetworkPlanID == nil {
		return errors.New("router_id or network_plan_id is required")
	}
	return nil
}

func normalizeVoucherServiceType(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return "pppoe"
	}
	return value
}

func buildVoucherUsername(prefix string, index int) string {
	safePrefix := strings.ToLower(strings.TrimSpace(prefix))
	safePrefix = strings.ReplaceAll(safePrefix, " ", "")
	if safePrefix == "" {
		safePrefix = "vcr"
	}
	return fmt.Sprintf("%s%s%02d", safePrefix, randomToken(4), index+1)
}

func randomToken(length int) string {
	const charset = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	if length <= 0 {
		length = 6
	}
	b := make([]byte, length)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			b[i] = charset[i%len(charset)]
			continue
		}
		b[i] = charset[n.Int64()]
	}
	return string(b)
}

func sanitizeVoucherBatch(input *models.VoucherBatch) *models.VoucherBatch {
	if input == nil {
		return nil
	}
	copy := *input
	if len(copy.Vouchers) > 0 {
		copy.Vouchers = append([]models.Voucher(nil), copy.Vouchers...)
		for i := range copy.Vouchers {
			copy.Vouchers[i] = *sanitizeVoucher(&copy.Vouchers[i])
		}
	}
	return &copy
}

func sanitizeVoucherPrefix(value string) string {
	value = strings.ToUpper(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, " ", "")
	if value == "" {
		return "VCR"
	}
	return value
}

func (s *voucherService) log(routerID *uuid.UUID, jobID *uuid.UUID, level string, action string, message string, requestPayload string, responsePayload string) {
	if s.logRepo == nil {
		return
	}
	_ = s.logRepo.Create(&models.ProvisioningLog{
		RouterID:          routerID,
		ProvisioningJobID: jobID,
		Level:             level,
		Action:            action,
		RequestPayload:    requestPayload,
		ResponsePayload:   responsePayload,
		Message:           message,
	})
}
