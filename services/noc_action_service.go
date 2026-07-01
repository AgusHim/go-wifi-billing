package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Agushim/go_wifi_billing/lib"
	"github.com/Agushim/go_wifi_billing/models"
	"github.com/Agushim/go_wifi_billing/repositories"
	"github.com/google/uuid"
)

var nocRouterCommandLimiter = struct {
	sync.Mutex
	lastRun map[uuid.UUID]time.Time
}{
	lastRun: map[uuid.UUID]time.Time{},
}

type NOCActionRequest struct {
	Action  string `json:"action"`
	Confirm bool   `json:"confirm"`
}

type NOCActionResult struct {
	Action              string              `json:"action"`
	Mode                string              `json:"mode"`
	ProvisioningJobID   *uuid.UUID          `json:"provisioning_job_id,omitempty"`
	ServiceAccountID    uuid.UUID           `json:"service_account_id"`
	RouterID            *uuid.UUID          `json:"router_id,omitempty"`
	Message             string              `json:"message"`
	RouterResponseItems []map[string]string `json:"router_response_items,omitempty"`
	ExecutedAt          time.Time           `json:"executed_at"`
}

type NOCActionService interface {
	RunServiceAccountAction(id uuid.UUID, request NOCActionRequest) (*NOCActionResult, error)
}

type nocActionService struct {
	serviceAccountSvc  ServiceAccountService
	serviceAccountRepo repositories.ServiceAccountRepository
	routerRepo         repositories.RouterRepository
	logRepo            repositories.ProvisioningLogRepository
}

func NewNOCActionService(
	serviceAccountSvc ServiceAccountService,
	serviceAccountRepo repositories.ServiceAccountRepository,
	routerRepo repositories.RouterRepository,
	logRepo repositories.ProvisioningLogRepository,
) NOCActionService {
	return &nocActionService{
		serviceAccountSvc:  serviceAccountSvc,
		serviceAccountRepo: serviceAccountRepo,
		routerRepo:         routerRepo,
		logRepo:            logRepo,
	}
}

func (s *nocActionService) RunServiceAccountAction(id uuid.UUID, request NOCActionRequest) (*NOCActionResult, error) {
	action := strings.TrimSpace(strings.ToLower(request.Action))
	if action == "" {
		return nil, errors.New("action is required")
	}
	if !request.Confirm {
		return nil, errors.New("confirmation is required for NOC runbook actions")
	}

	switch action {
	case "enable", "disable", "force_sync", "apply_profile":
		return s.enqueueProvisioningAction(id, action)
	case "reconnect", "clear_stale_session", "test_ping":
		return s.runRouterAction(id, action)
	default:
		return nil, errors.New("unsupported NOC action")
	}
}

func (s *nocActionService) enqueueProvisioningAction(id uuid.UUID, action string) (*NOCActionResult, error) {
	provisioningAction := map[string]string{
		"enable":        "unsuspend_account",
		"disable":       "suspend_account",
		"force_sync":    "create_account",
		"apply_profile": "change_plan",
	}[action]
	job, err := s.serviceAccountSvc.EnqueueAction(id, provisioningAction)
	if err != nil {
		return nil, err
	}
	account, _ := s.serviceAccountRepo.FindByID(id)
	var routerID *uuid.UUID
	if account != nil {
		routerID = serviceAccountRouterID(account)
	}
	s.audit(routerID, &job.ID, "noc_runbook_"+action, "provisioning job queued", map[string]string{
		"service_account_id":  id.String(),
		"provisioning_action": provisioningAction,
	})
	return &NOCActionResult{
		Action:            action,
		Mode:              "provisioning_job",
		ProvisioningJobID: &job.ID,
		ServiceAccountID:  id,
		RouterID:          routerID,
		Message:           "Provisioning job queued",
		ExecutedAt:        time.Now(),
	}, nil
}

func (s *nocActionService) runRouterAction(id uuid.UUID, action string) (*NOCActionResult, error) {
	account, err := s.serviceAccountRepo.FindByID(id)
	if err != nil {
		return nil, err
	}
	routerID := serviceAccountRouterID(account)
	if routerID == nil {
		return nil, errors.New("service account has no router mapping")
	}
	router, err := s.routerRepo.FindByID(*routerID)
	if err != nil {
		return nil, err
	}
	if nocRequireRouterTLS() && !router.UseTLS {
		return nil, errors.New("router TLS is required by NOC policy")
	}
	if err := throttleNOCRouterCommand(*routerID); err != nil {
		s.audit(routerID, nil, "noc_runbook_"+action, err.Error(), map[string]string{"service_account_id": id.String()})
		return nil, err
	}
	password, err := lib.DecryptSecret(router.PasswordEncrypted)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(password) == "" {
		return nil, errors.New("router credential is not configured")
	}

	runner, err := lib.NewMikrotikRunner(router.Host, router.Port, router.UseTLS, 5*time.Second, 5*time.Second)
	if err != nil {
		return nil, err
	}
	defer runner.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	login := runner.Login(ctx, router.Username, password)
	if !login.Success {
		return nil, login.Err
	}

	var items []map[string]string
	switch action {
	case "reconnect", "clear_stale_session":
		items, err = s.removeActiveSession(ctx, runner, account)
	case "test_ping":
		items, err = s.testPing(ctx, runner, account)
	default:
		err = errors.New("unsupported router action")
	}
	if err != nil {
		s.audit(routerID, nil, "noc_runbook_"+action, err.Error(), map[string]string{"service_account_id": id.String()})
		return nil, err
	}

	s.audit(routerID, nil, "noc_runbook_"+action, "router action succeeded", map[string]string{"service_account_id": id.String()})
	return &NOCActionResult{
		Action:              action,
		Mode:                "router_command",
		ServiceAccountID:    id,
		RouterID:            routerID,
		Message:             "Router action succeeded",
		RouterResponseItems: items,
		ExecutedAt:          time.Now(),
	}, nil
}

func throttleNOCRouterCommand(routerID uuid.UUID) error {
	interval := resolveNOCRouterCommandInterval()
	if interval <= 0 {
		return nil
	}
	nocRouterCommandLimiter.Lock()
	defer nocRouterCommandLimiter.Unlock()

	now := time.Now()
	lastRun, ok := nocRouterCommandLimiter.lastRun[routerID]
	if ok {
		nextAllowedAt := lastRun.Add(interval)
		if now.Before(nextAllowedAt) {
			return fmt.Errorf("router command rate limited, retry after %s", nextAllowedAt.Format(time.RFC3339))
		}
	}
	nocRouterCommandLimiter.lastRun[routerID] = now
	return nil
}

func resolveNOCRouterCommandInterval() time.Duration {
	value := strings.TrimSpace(os.Getenv("NOC_ROUTER_COMMAND_INTERVAL_SECONDS"))
	if value == "" {
		return 5 * time.Second
	}
	seconds, err := strconv.Atoi(value)
	if err != nil || seconds < 0 {
		return 5 * time.Second
	}
	return time.Duration(seconds) * time.Second
}

func (s *nocActionService) removeActiveSession(ctx context.Context, runner *lib.MikrotikRunner, account *models.ServiceAccount) ([]map[string]string, error) {
	printCommand, removeCommand, queryField, err := activeSessionCommands(account)
	if err != nil {
		return nil, err
	}
	find := runner.RunReadOnly(ctx, 1, printCommand, "?"+queryField+"="+account.Username)
	if !find.Success {
		return nil, find.Err
	}
	if len(find.Items) == 0 {
		return []map[string]string{
			{
				"status":  "noop",
				"message": "active session not found",
			},
		}, nil
	}
	var responses []map[string]string
	for _, item := range find.Items {
		remoteID := strings.TrimSpace(item[".id"])
		if remoteID == "" {
			continue
		}
		remove := runner.Run(ctx, removeCommand, "=.id="+remoteID)
		if !remove.Success {
			return responses, remove.Err
		}
		responses = append(responses, remove.Items...)
	}
	return responses, nil
}

func (s *nocActionService) testPing(ctx context.Context, runner *lib.MikrotikRunner, account *models.ServiceAccount) ([]map[string]string, error) {
	address := strings.TrimSpace(account.LastIPAddress)
	if address == "" {
		return nil, errors.New("service account has no last ip address")
	}
	result := runner.RunReadOnly(ctx, 1, "/ping", "=address="+address, "=count=4")
	if !result.Success {
		return nil, result.Err
	}
	return result.Items, nil
}

func activeSessionCommands(account *models.ServiceAccount) (string, string, string, error) {
	switch strings.TrimSpace(strings.ToLower(account.ServiceType)) {
	case "", "pppoe":
		return "/ppp/active/print", "/ppp/active/remove", "name", nil
	case "hotspot":
		return "/ip/hotspot/active/print", "/ip/hotspot/active/remove", "user", nil
	default:
		return "", "", "", errors.New("unsupported service account type")
	}
}

func (s *nocActionService) audit(routerID *uuid.UUID, jobID *uuid.UUID, action string, message string, payload map[string]string) {
	if s.logRepo == nil {
		return
	}
	payloadBytes, _ := json.Marshal(payload)
	_ = s.logRepo.Create(&models.ProvisioningLog{
		RouterID:          routerID,
		ProvisioningJobID: jobID,
		Level:             "info",
		Action:            action,
		Message:           message,
		RequestPayload:    string(payloadBytes),
	})
}
