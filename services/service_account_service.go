package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Agushim/go_wifi_billing/lib"
	"github.com/Agushim/go_wifi_billing/models"
	"github.com/Agushim/go_wifi_billing/repositories"
	"github.com/google/uuid"
)

type ServiceAccountService interface {
	Create(input *models.ServiceAccount) (*models.ServiceAccount, error)
	GetAll(subscriptionID string) ([]models.ServiceAccount, error)
	GetByID(id uuid.UUID) (*models.ServiceAccount, error)
	GetStatusHistory(id uuid.UUID, limit int) ([]models.ServiceStatusHistory, error)
	Update(id uuid.UUID, input *models.ServiceAccount) (*models.ServiceAccount, error)
	Delete(id uuid.UUID) error
	EnqueueAction(id uuid.UUID, action string) (*models.ProvisioningJob, error)
}

type serviceAccountService struct {
	repo        repositories.ServiceAccountRepository
	jobRepo     repositories.ProvisioningJobRepository
	logRepo     repositories.ProvisioningLogRepository
	routerRepo  repositories.RouterRepository
	historyRepo repositories.ServiceStatusHistoryRepository
}

func NewServiceAccountService(
	repo repositories.ServiceAccountRepository,
	jobRepo repositories.ProvisioningJobRepository,
	logRepo repositories.ProvisioningLogRepository,
	routerRepo repositories.RouterRepository,
	historyRepo repositories.ServiceStatusHistoryRepository,
) ServiceAccountService {
	return &serviceAccountService{
		repo:        repo,
		jobRepo:     jobRepo,
		logRepo:     logRepo,
		routerRepo:  routerRepo,
		historyRepo: historyRepo,
	}
}

func (s *serviceAccountService) Create(input *models.ServiceAccount) (*models.ServiceAccount, error) {
	if err := validateServiceAccount(input, true); err != nil {
		return nil, err
	}
	if strings.TrimSpace(input.Password) != "" {
		encrypted, err := lib.EncryptSecret(input.Password)
		if err != nil {
			return nil, err
		}
		input.PasswordEncrypted = encrypted
		input.Password = ""
	}
	if strings.TrimSpace(input.Status) == "" {
		input.Status = "pending"
	}
	if err := s.repo.Create(input); err != nil {
		return nil, err
	}
	s.recordStatusHistory(input.ID, "", input.Status, "created", "service_account", "service account created", nil, nil, nil)
	created, err := s.repo.FindByID(input.ID)
	if err != nil {
		return nil, err
	}
	return sanitizeServiceAccount(created), nil
}

func (s *serviceAccountService) GetAll(subscriptionID string) ([]models.ServiceAccount, error) {
	var accounts []models.ServiceAccount
	var err error
	if strings.TrimSpace(subscriptionID) == "" {
		accounts, err = s.repo.FindAll()
	} else {
		accounts, err = s.repo.FindBySubscriptionID(subscriptionID)
	}
	if err != nil {
		return nil, err
	}
	for i := range accounts {
		accounts[i] = *sanitizeServiceAccount(&accounts[i])
	}
	return accounts, nil
}

func (s *serviceAccountService) GetByID(id uuid.UUID) (*models.ServiceAccount, error) {
	item, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	return sanitizeServiceAccount(item), nil
}

func (s *serviceAccountService) GetStatusHistory(id uuid.UUID, limit int) ([]models.ServiceStatusHistory, error) {
	return s.historyRepo.FindByServiceAccountID(id, limit)
}

func (s *serviceAccountService) Update(id uuid.UUID, input *models.ServiceAccount) (*models.ServiceAccount, error) {
	if err := validateServiceAccount(input, false); err != nil {
		return nil, err
	}
	existing, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	existing.SubscriptionID = input.SubscriptionID
	existing.RouterID = input.RouterID
	existing.NetworkPlanID = input.NetworkPlanID
	existing.ServiceType = input.ServiceType
	existing.Username = input.Username
	existing.RemoteName = input.RemoteName
	existing.RemoteID = input.RemoteID
	previousStatus := existing.Status
	existing.Status = input.Status
	if strings.TrimSpace(input.Password) != "" {
		encrypted, err := lib.EncryptSecret(input.Password)
		if err != nil {
			return nil, err
		}
		existing.PasswordEncrypted = encrypted
	}
	if err := s.repo.Update(existing); err != nil {
		return nil, err
	}
	if !strings.EqualFold(strings.TrimSpace(previousStatus), strings.TrimSpace(existing.Status)) {
		s.recordStatusHistory(existing.ID, previousStatus, existing.Status, "manual_update", "service_account", "status updated from service account form", nil, nil, nil)
	}
	updated, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	return sanitizeServiceAccount(updated), nil
}

func (s *serviceAccountService) Delete(id uuid.UUID) error {
	return s.repo.Delete(id)
}

func (s *serviceAccountService) EnqueueAction(id uuid.UUID, action string) (*models.ProvisioningJob, error) {
	account, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if err := validateProvisioningAction(action); err != nil {
		return nil, err
	}

	payloadMap := map[string]string{
		"service_account_id": account.ID.String(),
		"username":           account.Username,
		"service_type":       account.ServiceType,
	}
	payloadBytes, _ := json.Marshal(payloadMap)

	job := &models.ProvisioningJob{
		EntityType: "service_account",
		EntityID:   &account.ID,
		Action:     action,
		Payload:    string(payloadBytes),
		Status:     "pending",
	}
	if err := s.jobRepo.Create(job); err != nil {
		return nil, err
	}

	s.log(account.RouterID, &job.ID, "info", action, fmt.Sprintf("job queued for %s", account.Username), string(payloadBytes), "")

	if provisioningEnabled() {
		go s.processJob(job.ID)
	}

	return job, nil
}

func (s *serviceAccountService) processJob(jobID uuid.UUID) {
	job, err := s.jobRepo.FindByID(jobID)
	if err != nil || job == nil {
		return
	}
	if job.EntityID == nil {
		job.Status = "failed"
		job.ErrorMessage = "job has no entity id"
		now := time.Now()
		job.ExecutedAt = &now
		_ = s.jobRepo.Update(job)
		return
	}

	account, err := s.repo.FindByID(*job.EntityID)
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

	if err := s.executeProvisioningAction(account, job); err != nil {
		job.Status = "failed"
		job.ErrorMessage = err.Error()
		_ = s.jobRepo.Update(job)
		s.log(account.RouterID, &job.ID, "error", job.Action, err.Error(), job.Payload, "")

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

func (s *serviceAccountService) executeProvisioningAction(account *models.ServiceAccount, job *models.ProvisioningJob) error {
	router, err := resolveProvisioningRouter(account)
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

	accountPassword, err := lib.DecryptSecret(account.PasswordEncrypted)
	if err != nil {
		return err
	}
	if job.Action == "create_account" && strings.TrimSpace(accountPassword) == "" {
		return errors.New("service account password is required for create_account")
	}

	var requestWords []string
	switch account.ServiceType {
	case "pppoe", "":
		requestWords, err = pppoeRequestWords(account, accountPassword, job.Action)
	case "hotspot":
		requestWords, err = hotspotRequestWords(account, accountPassword, job.Action)
	default:
		return errors.New("unsupported service account type")
	}
	if err != nil {
		return err
	}

	if job.Action == "suspend_account" || job.Action == "unsuspend_account" || job.Action == "terminate_account" || job.Action == "change_plan" {
		requestWords, err = s.expandSetOrRemoveWords(client, account, requestWords, job.Action)
		if err != nil {
			return err
		}
	}

	response, err := client.Run(requestWords...)
	responseBytes, _ := json.Marshal(response)
	if err != nil {
		return err
	}

	previousStatus := account.Status
	switch job.Action {
	case "create_account", "unsuspend_account", "change_plan":
		account.Status = "active"
	case "suspend_account":
		account.Status = "suspended"
	case "terminate_account":
		account.Status = "terminated"
	}
	account.RemoteName = account.Username
	account.LastSyncedAt = job.ExecutedAt
	_ = s.repo.Update(account)
	s.recordStatusHistory(account.ID, previousStatus, account.Status, job.Action, "provisioning_job", "provisioning command succeeded", &job.ID, nil, nil)

	s.log(account.RouterID, &job.ID, "info", job.Action, "provisioning command succeeded", strings.Join(requestWords, " "), string(responseBytes))
	return nil
}

func (s *serviceAccountService) expandSetOrRemoveWords(client *lib.MikrotikClient, account *models.ServiceAccount, requestWords []string, action string) ([]string, error) {
	findCommand, err := findCommandForAccount(account)
	if err != nil {
		return nil, err
	}
	result, err := client.Run(findCommand, "?name="+account.Username)
	if err != nil {
		return nil, err
	}
	if len(result) == 0 || strings.TrimSpace(result[0][".id"]) == "" {
		return nil, errors.New("service account not found on router")
	}

	words := append([]string{}, requestWords...)
	words = append(words, "=.id="+result[0][".id"])
	return words, nil
}

func pppoeRequestWords(account *models.ServiceAccount, password string, action string) ([]string, error) {
	switch action {
	case "create_account":
		words := []string{
			"/ppp/secret/add",
			"=name=" + account.Username,
			"=password=" + password,
			"=service=pppoe",
		}
		if account.NetworkPlan != nil && strings.TrimSpace(account.NetworkPlan.MikrotikProfileName) != "" {
			words = append(words, "=profile="+account.NetworkPlan.MikrotikProfileName)
		}
		if account.NetworkPlan != nil && strings.TrimSpace(account.NetworkPlan.AddressPool) != "" {
			words = append(words, "=remote-address="+account.NetworkPlan.AddressPool)
		}
		return words, nil
	case "suspend_account":
		return []string{"/ppp/secret/set", "=disabled=yes"}, nil
	case "unsuspend_account":
		return []string{"/ppp/secret/set", "=disabled=no"}, nil
	case "terminate_account":
		return []string{"/ppp/secret/remove"}, nil
	case "change_plan":
		words := []string{"/ppp/secret/set"}
		if account.NetworkPlan != nil && strings.TrimSpace(account.NetworkPlan.MikrotikProfileName) != "" {
			words = append(words, "=profile="+account.NetworkPlan.MikrotikProfileName)
		}
		return words, nil
	default:
		return nil, errors.New("unsupported provisioning action")
	}
}

func hotspotRequestWords(account *models.ServiceAccount, password string, action string) ([]string, error) {
	switch action {
	case "create_account":
		words := []string{
			"/ip/hotspot/user/add",
			"=name=" + account.Username,
			"=password=" + password,
		}
		if account.NetworkPlan != nil && strings.TrimSpace(account.NetworkPlan.MikrotikProfileName) != "" {
			words = append(words, "=profile="+account.NetworkPlan.MikrotikProfileName)
		}
		return words, nil
	case "suspend_account":
		return []string{"/ip/hotspot/user/set", "=disabled=yes"}, nil
	case "unsuspend_account":
		return []string{"/ip/hotspot/user/set", "=disabled=no"}, nil
	case "terminate_account":
		return []string{"/ip/hotspot/user/remove"}, nil
	case "change_plan":
		words := []string{"/ip/hotspot/user/set"}
		if account.NetworkPlan != nil && strings.TrimSpace(account.NetworkPlan.MikrotikProfileName) != "" {
			words = append(words, "=profile="+account.NetworkPlan.MikrotikProfileName)
		}
		return words, nil
	default:
		return nil, errors.New("unsupported provisioning action")
	}
}

func findCommandForAccount(account *models.ServiceAccount) (string, error) {
	switch account.ServiceType {
	case "pppoe", "":
		return "/ppp/secret/print", nil
	case "hotspot":
		return "/ip/hotspot/user/print", nil
	default:
		return "", errors.New("unsupported service account type")
	}
}

func validateProvisioningAction(action string) error {
	switch strings.TrimSpace(action) {
	case "create_account", "suspend_account", "unsuspend_account", "terminate_account", "change_plan":
		return nil
	default:
		return errors.New("unsupported provisioning action")
	}
}

func validateServiceAccount(input *models.ServiceAccount, passwordRequired bool) error {
	if input == nil {
		return errors.New("invalid service account payload")
	}
	if input.SubscriptionID == uuid.Nil {
		return errors.New("subscription_id is required")
	}
	if strings.TrimSpace(input.Username) == "" {
		return errors.New("username is required")
	}
	if strings.TrimSpace(input.ServiceType) == "" {
		input.ServiceType = "pppoe"
	}
	if passwordRequired && strings.TrimSpace(input.Password) == "" {
		return errors.New("password is required")
	}
	if strings.TrimSpace(input.Status) == "" {
		input.Status = "pending"
	}
	return nil
}

func sanitizeServiceAccount(input *models.ServiceAccount) *models.ServiceAccount {
	if input == nil {
		return nil
	}
	copy := *input
	copy.HasPassword = strings.TrimSpace(copy.PasswordEncrypted) != ""
	copy.Password = ""
	return &copy
}

func resolveProvisioningRouter(account *models.ServiceAccount) (*models.Router, error) {
	if account == nil {
		return nil, errors.New("service account is nil")
	}
	if account.Router != nil {
		return account.Router, nil
	}
	if account.NetworkPlan != nil && account.NetworkPlan.Router != nil {
		return account.NetworkPlan.Router, nil
	}
	return nil, errors.New("service account has no router mapping")
}

func provisioningEnabled() bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv("FEATURE_PROVISIONING_ENABLED")))
	return value == "1" || value == "true" || value == "yes"
}

func (s *serviceAccountService) log(routerID *uuid.UUID, jobID *uuid.UUID, level string, action string, message string, requestPayload string, responsePayload string) {
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

func (s *serviceAccountService) recordStatusHistory(
	serviceAccountID uuid.UUID,
	previousStatus string,
	newStatus string,
	action string,
	source string,
	note string,
	provisioningJobID *uuid.UUID,
	billID *uuid.UUID,
	paymentID *uuid.UUID,
) {
	if s.historyRepo == nil {
		return
	}
	if strings.TrimSpace(previousStatus) == strings.TrimSpace(newStatus) {
		return
	}
	_ = s.historyRepo.Create(&models.ServiceStatusHistory{
		ServiceAccountID:  serviceAccountID,
		PreviousStatus:    previousStatus,
		NewStatus:         newStatus,
		Action:            action,
		Source:            source,
		Note:              note,
		ProvisioningJobID: provisioningJobID,
		BillID:            billID,
		PaymentID:         paymentID,
	})
}
