package services

import (
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

type RouterTestResult struct {
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	LatencyMS int64  `json:"latency_ms"`
	Identity  string `json:"identity"`
	Version   string `json:"version"`
}

type RouterResourceResponse struct {
	Kind  string              `json:"kind"`
	Items []map[string]string `json:"items"`
}

type RouterImportPreview struct {
	RouterID        uuid.UUID                    `json:"router_id"`
	RouterName      string                       `json:"router_name"`
	Mode            string                       `json:"mode"`
	NetworkPlans    []RouterImportPlanCandidate  `json:"network_plans"`
	ServiceAccounts []RouterImportAccountPreview `json:"service_accounts"`
	Summary         RouterImportSummary          `json:"summary"`
	GeneratedAt     time.Time                    `json:"generated_at"`
}

type RouterImportBatchResponse struct {
	ID                      uuid.UUID               `json:"id"`
	RouterID                uuid.UUID               `json:"router_id"`
	RouterName              string                  `json:"router_name"`
	Mode                    string                  `json:"mode"`
	Status                  string                  `json:"status"`
	TotalNetworkPlans       int                     `json:"total_network_plans"`
	NewNetworkPlans         int                     `json:"new_network_plans"`
	ExistingNetworkPlans    int                     `json:"existing_network_plans"`
	TotalServiceAccounts    int                     `json:"total_service_accounts"`
	NewServiceAccounts      int                     `json:"new_service_accounts"`
	ExistingServiceAccounts int                     `json:"existing_service_accounts"`
	Items                   []RouterImportBatchItem `json:"items,omitempty"`
	CreatedAt               time.Time               `json:"created_at"`
}

type RouterImportBatchItem struct {
	ID                       uuid.UUID  `json:"id"`
	ItemType                 string     `json:"item_type"`
	ServiceType              string     `json:"service_type"`
	Username                 string     `json:"username"`
	RemoteID                 string     `json:"remote_id"`
	ProfileName              string     `json:"profile_name"`
	AddressPool              string     `json:"address_pool"`
	RemoteStatus             string     `json:"remote_status"`
	SuggestedName            string     `json:"suggested_name"`
	ExistingNetworkPlanID    *uuid.UUID `json:"existing_network_plan_id,omitempty"`
	ExistingNetworkPlan      string     `json:"existing_network_plan,omitempty"`
	ExistingServiceAccountID *uuid.UUID `json:"existing_service_account_id,omitempty"`
	ExistingServiceAccount   string     `json:"existing_service_account,omitempty"`
	MatchedNetworkPlanID     *uuid.UUID `json:"matched_network_plan_id,omitempty"`
	MatchedNetworkPlan       string     `json:"matched_network_plan,omitempty"`
	Conflict                 bool       `json:"conflict"`
	RecommendedAction        string     `json:"recommended_action"`
	StageStatus              string     `json:"stage_status"`
	Note                     string     `json:"note"`
}

type RouterHealthCheckSummary struct {
	Total     int       `json:"total"`
	Healthy   int       `json:"healthy"`
	Failed    int       `json:"failed"`
	CheckedAt time.Time `json:"checked_at"`
}

type RouterImportSummary struct {
	TotalNetworkPlans       int `json:"total_network_plans"`
	NewNetworkPlans         int `json:"new_network_plans"`
	ExistingNetworkPlans    int `json:"existing_network_plans"`
	TotalServiceAccounts    int `json:"total_service_accounts"`
	NewServiceAccounts      int `json:"new_service_accounts"`
	ExistingServiceAccounts int `json:"existing_service_accounts"`
}

type RouterImportPlanCandidate struct {
	ServiceType           string     `json:"service_type"`
	ProfileName           string     `json:"profile_name"`
	AddressPool           string     `json:"address_pool"`
	SuggestedName         string     `json:"suggested_name"`
	ExistingNetworkPlanID *uuid.UUID `json:"existing_network_plan_id,omitempty"`
	ExistingNetworkPlan   string     `json:"existing_network_plan,omitempty"`
	Conflict              bool       `json:"conflict"`
	RecommendedAction     string     `json:"recommended_action"`
}

type RouterImportAccountPreview struct {
	ServiceType              string     `json:"service_type"`
	Username                 string     `json:"username"`
	RemoteID                 string     `json:"remote_id"`
	ProfileName              string     `json:"profile_name"`
	RemoteStatus             string     `json:"remote_status"`
	Comment                  string     `json:"comment"`
	ExistingServiceAccountID *uuid.UUID `json:"existing_service_account_id,omitempty"`
	ExistingServiceAccount   string     `json:"existing_service_account,omitempty"`
	MatchedNetworkPlanID     *uuid.UUID `json:"matched_network_plan_id,omitempty"`
	MatchedNetworkPlan       string     `json:"matched_network_plan,omitempty"`
	Conflict                 bool       `json:"conflict"`
	RecommendedAction        string     `json:"recommended_action"`
}

type RouterService interface {
	Create(input *models.Router) (*models.Router, error)
	GetAll() ([]models.Router, error)
	GetByID(id uuid.UUID) (*models.Router, error)
	Update(id uuid.UUID, input *models.Router) (*models.Router, error)
	Delete(id uuid.UUID) error
	TestConnection(id uuid.UUID) (*RouterTestResult, error)
	FetchResources(id uuid.UUID, kind string) (*RouterResourceResponse, error)
	RunHealthCheckAll() (*RouterHealthCheckSummary, error)
	StartHealthCheckScheduler()
	PreviewImport(id uuid.UUID, mode string) (*RouterImportPreview, error)
	StageImport(id uuid.UUID, mode string) (*RouterImportBatchResponse, error)
	ListImportBatches(id uuid.UUID) ([]RouterImportBatchResponse, error)
	GetImportBatch(id uuid.UUID) (*RouterImportBatchResponse, error)
}

type routerService struct {
	repo               repositories.RouterRepository
	logRepo            repositories.ProvisioningLogRepository
	networkPlanRepo    repositories.NetworkPlanRepository
	serviceAccountRepo repositories.ServiceAccountRepository
	importBatchRepo    repositories.RouterImportBatchRepository
	importItemRepo     repositories.RouterImportItemRepository
}

func NewRouterService(
	repo repositories.RouterRepository,
	logRepo repositories.ProvisioningLogRepository,
	networkPlanRepo repositories.NetworkPlanRepository,
	serviceAccountRepo repositories.ServiceAccountRepository,
	importBatchRepo repositories.RouterImportBatchRepository,
	importItemRepo repositories.RouterImportItemRepository,
) RouterService {
	return &routerService{
		repo:               repo,
		logRepo:            logRepo,
		networkPlanRepo:    networkPlanRepo,
		serviceAccountRepo: serviceAccountRepo,
		importBatchRepo:    importBatchRepo,
		importItemRepo:     importItemRepo,
	}
}

func (s *routerService) Create(input *models.Router) (*models.Router, error) {
	if err := validateRouterInput(input, true); err != nil {
		return nil, err
	}
	encrypted, err := lib.EncryptSecret(input.Password)
	if err != nil {
		return nil, err
	}
	input.PasswordEncrypted = encrypted
	input.Password = ""
	if strings.TrimSpace(input.Status) == "" {
		input.Status = "unknown"
	}
	if err := s.repo.Create(input); err != nil {
		return nil, err
	}
	return sanitizeRouter(input), nil
}

func (s *routerService) GetAll() ([]models.Router, error) {
	routers, err := s.repo.FindAll()
	if err != nil {
		return nil, err
	}
	for i := range routers {
		routers[i] = *sanitizeRouter(&routers[i])
	}
	return routers, nil
}

func (s *routerService) GetByID(id uuid.UUID) (*models.Router, error) {
	router, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	return sanitizeRouter(router), nil
}

func (s *routerService) Update(id uuid.UUID, input *models.Router) (*models.Router, error) {
	if err := validateRouterInput(input, false); err != nil {
		return nil, err
	}
	existing, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}

	existing.Name = input.Name
	existing.Host = input.Host
	existing.Port = input.Port
	existing.Username = input.Username
	existing.APIType = input.APIType
	existing.UseTLS = input.UseTLS
	existing.Location = input.Location
	existing.Status = input.Status
	existing.LastError = input.LastError
	existing.LastSeenAt = input.LastSeenAt
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
	return sanitizeRouter(existing), nil
}

func (s *routerService) Delete(id uuid.UUID) error {
	return s.repo.Delete(id)
}

func (s *routerService) TestConnection(id uuid.UUID) (*RouterTestResult, error) {
	router, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	password, err := lib.DecryptSecret(router.PasswordEncrypted)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(password) == "" {
		return nil, errors.New("router password is not configured")
	}

	start := time.Now()
	client, err := lib.NewMikrotikClient(router.Host, router.Port, router.UseTLS, 5*time.Second)
	if err != nil {
		s.markRouterError(router, "unreachable", err.Error())
		s.log(router.ID, "error", "test_connection", "connection failed")
		return nil, err
	}
	defer client.Close()

	if err := client.Login(router.Username, password); err != nil {
		s.markRouterError(router, "auth_failed", err.Error())
		s.log(router.ID, "error", "test_connection", "authentication failed")
		return nil, err
	}

	identity := ""
	version := ""
	if items, err := client.Run("/system/identity/print"); err == nil && len(items) > 0 {
		identity = items[0]["name"]
	}
	if items, err := client.Run("/system/resource/print"); err == nil && len(items) > 0 {
		version = items[0]["version"]
	}

	now := time.Now()
	router.Status = "connected"
	router.LastSeenAt = &now
	router.LastError = ""
	_ = s.repo.Update(router)
	s.log(router.ID, "info", "test_connection", "connection test succeeded")

	return &RouterTestResult{
		Success:   true,
		Message:   "Connection successful",
		LatencyMS: time.Since(start).Milliseconds(),
		Identity:  identity,
		Version:   version,
	}, nil
}

func (s *routerService) FetchResources(id uuid.UUID, kind string) (*RouterResourceResponse, error) {
	router, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	command, err := resourceCommand(kind)
	if err != nil {
		return nil, err
	}
	password, err := lib.DecryptSecret(router.PasswordEncrypted)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(password) == "" {
		return nil, errors.New("router password is not configured")
	}

	client, err := lib.NewMikrotikClient(router.Host, router.Port, router.UseTLS, 5*time.Second)
	if err != nil {
		s.markRouterError(router, "unreachable", err.Error())
		s.log(router.ID, "error", "fetch_resource", fmt.Sprintf("resource fetch failed for %s", kind))
		return nil, err
	}
	defer client.Close()

	if err := client.Login(router.Username, password); err != nil {
		s.markRouterError(router, "auth_failed", err.Error())
		s.log(router.ID, "error", "fetch_resource", fmt.Sprintf("authentication failed for %s", kind))
		return nil, err
	}

	items, err := client.Run(command)
	if err != nil {
		s.markRouterError(router, router.Status, err.Error())
		s.log(router.ID, "error", "fetch_resource", err.Error())
		return nil, err
	}

	now := time.Now()
	router.Status = "connected"
	router.LastSeenAt = &now
	router.LastError = ""
	_ = s.repo.Update(router)
	s.log(router.ID, "info", "fetch_resource", fmt.Sprintf("resource fetch succeeded for %s", kind))

	return &RouterResourceResponse{
		Kind:  kind,
		Items: items,
	}, nil
}

func (s *routerService) RunHealthCheckAll() (*RouterHealthCheckSummary, error) {
	routers, err := s.repo.FindAll()
	if err != nil {
		return nil, err
	}

	summary := &RouterHealthCheckSummary{
		Total:     len(routers),
		CheckedAt: time.Now(),
	}
	for _, router := range routers {
		if _, err := s.TestConnection(router.ID); err != nil {
			summary.Failed++
			continue
		}
		summary.Healthy++
	}

	s.log(uuid.Nil, "info", "router_health_check", fmt.Sprintf("health check finished: healthy=%d failed=%d", summary.Healthy, summary.Failed))
	return summary, nil
}

func (s *routerService) StartHealthCheckScheduler() {
	if !routerSyncEnabled() {
		return
	}

	interval := resolveRouterHealthCheckInterval()
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			if _, err := s.RunHealthCheckAll(); err != nil {
				s.log(uuid.Nil, "error", "router_health_check", err.Error())
			}
		}
	}()
}

func (s *routerService) PreviewImport(id uuid.UUID, mode string) (*RouterImportPreview, error) {
	return s.buildImportPreview(id, mode)
}

func (s *routerService) StageImport(id uuid.UUID, mode string) (*RouterImportBatchResponse, error) {
	preview, err := s.buildImportPreview(id, mode)
	if err != nil {
		return nil, err
	}

	batch := &models.RouterImportBatch{
		RouterID:                preview.RouterID,
		Mode:                    preview.Mode,
		Status:                  "staged",
		TotalNetworkPlans:       preview.Summary.TotalNetworkPlans,
		NewNetworkPlans:         preview.Summary.NewNetworkPlans,
		ExistingNetworkPlans:    preview.Summary.ExistingNetworkPlans,
		TotalServiceAccounts:    preview.Summary.TotalServiceAccounts,
		NewServiceAccounts:      preview.Summary.NewServiceAccounts,
		ExistingServiceAccounts: preview.Summary.ExistingServiceAccounts,
	}
	if err := s.importBatchRepo.Create(batch); err != nil {
		return nil, err
	}

	items := make([]models.RouterImportItem, 0, len(preview.NetworkPlans)+len(preview.ServiceAccounts))
	for _, item := range preview.NetworkPlans {
		items = append(items, models.RouterImportItem{
			BatchID:               batch.ID,
			ItemType:              "network_plan",
			ServiceType:           item.ServiceType,
			ProfileName:           item.ProfileName,
			AddressPool:           item.AddressPool,
			SuggestedName:         item.SuggestedName,
			ExistingNetworkPlanID: item.ExistingNetworkPlanID,
			Conflict:              item.Conflict,
			RecommendedAction:     item.RecommendedAction,
			StageStatus:           "staged",
			Note:                  buildPlanImportNote(item),
		})
	}
	for _, item := range preview.ServiceAccounts {
		items = append(items, models.RouterImportItem{
			BatchID:                  batch.ID,
			ItemType:                 "service_account",
			ServiceType:              item.ServiceType,
			Username:                 item.Username,
			RemoteID:                 item.RemoteID,
			ProfileName:              item.ProfileName,
			RemoteStatus:             item.RemoteStatus,
			ExistingServiceAccountID: item.ExistingServiceAccountID,
			MatchedNetworkPlanID:     item.MatchedNetworkPlanID,
			Conflict:                 item.Conflict,
			RecommendedAction:        item.RecommendedAction,
			StageStatus:              "staged",
			Note:                     item.Comment,
		})
	}
	if err := s.importItemRepo.CreateMany(items); err != nil {
		return nil, err
	}

	s.log(id, "info", "stage_import", fmt.Sprintf("router import staged for mode %s", preview.Mode))
	return s.GetImportBatch(batch.ID)
}

func (s *routerService) ListImportBatches(id uuid.UUID) ([]RouterImportBatchResponse, error) {
	items, err := s.importBatchRepo.FindByRouterID(id)
	if err != nil {
		return nil, err
	}
	responses := make([]RouterImportBatchResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, RouterImportBatchResponse{
			ID:                      item.ID,
			RouterID:                item.RouterID,
			RouterName:              resolveRouterName(item.Router),
			Mode:                    item.Mode,
			Status:                  item.Status,
			TotalNetworkPlans:       item.TotalNetworkPlans,
			NewNetworkPlans:         item.NewNetworkPlans,
			ExistingNetworkPlans:    item.ExistingNetworkPlans,
			TotalServiceAccounts:    item.TotalServiceAccounts,
			NewServiceAccounts:      item.NewServiceAccounts,
			ExistingServiceAccounts: item.ExistingServiceAccounts,
			CreatedAt:               item.CreatedAt,
		})
	}
	return responses, nil
}

func (s *routerService) GetImportBatch(id uuid.UUID) (*RouterImportBatchResponse, error) {
	batch, err := s.importBatchRepo.FindByID(id)
	if err != nil {
		return nil, err
	}
	response := &RouterImportBatchResponse{
		ID:                      batch.ID,
		RouterID:                batch.RouterID,
		RouterName:              resolveRouterName(batch.Router),
		Mode:                    batch.Mode,
		Status:                  batch.Status,
		TotalNetworkPlans:       batch.TotalNetworkPlans,
		NewNetworkPlans:         batch.NewNetworkPlans,
		ExistingNetworkPlans:    batch.ExistingNetworkPlans,
		TotalServiceAccounts:    batch.TotalServiceAccounts,
		NewServiceAccounts:      batch.NewServiceAccounts,
		ExistingServiceAccounts: batch.ExistingServiceAccounts,
		CreatedAt:               batch.CreatedAt,
		Items:                   make([]RouterImportBatchItem, 0, len(batch.Items)),
	}
	for _, item := range batch.Items {
		response.Items = append(response.Items, RouterImportBatchItem{
			ID:                       item.ID,
			ItemType:                 item.ItemType,
			ServiceType:              item.ServiceType,
			Username:                 item.Username,
			RemoteID:                 item.RemoteID,
			ProfileName:              item.ProfileName,
			AddressPool:              item.AddressPool,
			RemoteStatus:             item.RemoteStatus,
			SuggestedName:            item.SuggestedName,
			ExistingNetworkPlanID:    item.ExistingNetworkPlanID,
			ExistingNetworkPlan:      resolveNetworkPlanName(item.ExistingNetworkPlan),
			ExistingServiceAccountID: item.ExistingServiceAccountID,
			ExistingServiceAccount:   resolveServiceAccountName(item.ExistingServiceAccount),
			MatchedNetworkPlanID:     item.MatchedNetworkPlanID,
			MatchedNetworkPlan:       resolveNetworkPlanName(item.MatchedNetworkPlan),
			Conflict:                 item.Conflict,
			RecommendedAction:        item.RecommendedAction,
			StageStatus:              item.StageStatus,
			Note:                     item.Note,
		})
	}
	return response, nil
}

func (s *routerService) buildImportPreview(id uuid.UUID, mode string) (*RouterImportPreview, error) {
	router, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}

	mode = normalizeImportMode(mode)
	if mode == "" {
		return nil, errors.New("unsupported import mode")
	}

	password, err := lib.DecryptSecret(router.PasswordEncrypted)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(password) == "" {
		return nil, errors.New("router password is not configured")
	}

	client, err := lib.NewMikrotikClient(router.Host, router.Port, router.UseTLS, 5*time.Second)
	if err != nil {
		s.markRouterError(router, "unreachable", err.Error())
		s.log(router.ID, "error", "preview_import", "connection failed")
		return nil, err
	}
	defer client.Close()

	if err := client.Login(router.Username, password); err != nil {
		s.markRouterError(router, "auth_failed", err.Error())
		s.log(router.ID, "error", "preview_import", "authentication failed")
		return nil, err
	}

	existingPlans, err := s.networkPlanRepo.FindAll()
	if err != nil {
		return nil, err
	}
	existingAccounts, err := s.serviceAccountRepo.FindAll()
	if err != nil {
		return nil, err
	}

	planCandidates := make(map[string]RouterImportPlanCandidate)
	accountCandidates := make([]RouterImportAccountPreview, 0)
	planLookup := make(map[string]RouterImportPlanCandidate)

	if mode == "all" || mode == "pppoe" {
		pppProfiles, err := client.Run("/ppp/profile/print")
		if err != nil {
			return nil, err
		}
		pppSecrets, err := client.Run("/ppp/secret/print")
		if err != nil {
			return nil, err
		}
		appendPlanCandidates(planCandidates, planLookup, buildPPPoEPlanCandidates(router, existingPlans, pppProfiles)...)
		accountCandidates = append(accountCandidates, buildPPPoEAccountCandidates(router, existingAccounts, planLookup, pppSecrets)...)
	}

	if mode == "all" || mode == "hotspot" {
		hotspotProfiles, err := client.Run("/ip/hotspot/user/profile/print")
		if err != nil {
			return nil, err
		}
		hotspotUsers, err := client.Run("/ip/hotspot/user/print")
		if err != nil {
			return nil, err
		}
		appendPlanCandidates(planCandidates, planLookup, buildHotspotPlanCandidates(router, existingPlans, hotspotProfiles)...)
		accountCandidates = append(accountCandidates, buildHotspotAccountCandidates(router, existingAccounts, planLookup, hotspotUsers)...)
	}

	plans := make([]RouterImportPlanCandidate, 0, len(planCandidates))
	for _, candidate := range planCandidates {
		plans = append(plans, candidate)
	}

	summary := RouterImportSummary{
		TotalNetworkPlans:    len(plans),
		TotalServiceAccounts: len(accountCandidates),
	}
	for _, item := range plans {
		if item.Conflict {
			summary.ExistingNetworkPlans++
		} else {
			summary.NewNetworkPlans++
		}
	}
	for _, item := range accountCandidates {
		if item.Conflict {
			summary.ExistingServiceAccounts++
		} else {
			summary.NewServiceAccounts++
		}
	}

	now := time.Now()
	router.Status = "connected"
	router.LastSeenAt = &now
	router.LastError = ""
	_ = s.repo.Update(router)
	s.log(router.ID, "info", "preview_import", fmt.Sprintf("router import preview generated for mode %s", mode))

	return &RouterImportPreview{
		RouterID:        router.ID,
		RouterName:      router.Name,
		Mode:            mode,
		NetworkPlans:    plans,
		ServiceAccounts: accountCandidates,
		Summary:         summary,
		GeneratedAt:     now,
	}, nil
}

func validateRouterInput(input *models.Router, passwordRequired bool) error {
	if input == nil {
		return errors.New("invalid router payload")
	}
	if strings.TrimSpace(input.Name) == "" {
		return errors.New("router name is required")
	}
	if strings.TrimSpace(input.Host) == "" {
		return errors.New("router host is required")
	}
	if input.Port <= 0 {
		input.Port = 8728
	}
	if strings.TrimSpace(input.APIType) == "" {
		input.APIType = "routeros"
	}
	if strings.TrimSpace(input.Status) == "" {
		input.Status = "unknown"
	}
	if passwordRequired && strings.TrimSpace(input.Password) == "" {
		return errors.New("router password is required")
	}
	return nil
}

func sanitizeRouter(router *models.Router) *models.Router {
	if router == nil {
		return nil
	}
	copy := *router
	copy.HasPassword = strings.TrimSpace(copy.PasswordEncrypted) != ""
	copy.Password = ""
	return &copy
}

func resourceCommand(kind string) (string, error) {
	switch strings.TrimSpace(kind) {
	case "ppp-profiles":
		return "/ppp/profile/print", nil
	case "ppp-secrets":
		return "/ppp/secret/print", nil
	case "ip-pools":
		return "/ip/pool/print", nil
	case "hotspot-profiles":
		return "/ip/hotspot/user/profile/print", nil
	case "hotspot-users":
		return "/ip/hotspot/user/print", nil
	case "hotspot-servers":
		return "/ip/hotspot/print", nil
	default:
		return "", errors.New("unsupported resource kind")
	}
}

func normalizeImportMode(mode string) string {
	value := strings.TrimSpace(strings.ToLower(mode))
	switch value {
	case "", "all":
		return "all"
	case "pppoe", "ppp", "pppoe-only":
		return "pppoe"
	case "hotspot", "hotspot-only":
		return "hotspot"
	default:
		return ""
	}
}

func appendPlanCandidates(
	target map[string]RouterImportPlanCandidate,
	lookup map[string]RouterImportPlanCandidate,
	items ...RouterImportPlanCandidate,
) {
	for _, item := range items {
		key := planCandidateKey(item.ServiceType, item.ProfileName)
		target[key] = item
		lookup[key] = item
	}
}

func buildPPPoEPlanCandidates(
	router *models.Router,
	existingPlans []models.NetworkPlan,
	profiles []map[string]string,
) []RouterImportPlanCandidate {
	items := make([]RouterImportPlanCandidate, 0, len(profiles))
	for _, profile := range profiles {
		profileName := strings.TrimSpace(profile["name"])
		if profileName == "" {
			continue
		}
		existing := matchExistingNetworkPlan(existingPlans, router.ID, "pppoe", profileName)
		candidate := RouterImportPlanCandidate{
			ServiceType:       "pppoe",
			ProfileName:       profileName,
			AddressPool:       strings.TrimSpace(profile["remote-address"]),
			SuggestedName:     buildImportPlanName(router.Name, profileName, "pppoe"),
			Conflict:          existing != nil,
			RecommendedAction: "create_network_plan",
		}
		if existing != nil {
			candidate.ExistingNetworkPlanID = &existing.ID
			candidate.ExistingNetworkPlan = existing.Name
			candidate.RecommendedAction = "skip_existing"
		}
		items = append(items, candidate)
	}
	return items
}

func buildHotspotPlanCandidates(
	router *models.Router,
	existingPlans []models.NetworkPlan,
	profiles []map[string]string,
) []RouterImportPlanCandidate {
	items := make([]RouterImportPlanCandidate, 0, len(profiles))
	for _, profile := range profiles {
		profileName := strings.TrimSpace(profile["name"])
		if profileName == "" {
			continue
		}
		existing := matchExistingNetworkPlan(existingPlans, router.ID, "hotspot", profileName)
		candidate := RouterImportPlanCandidate{
			ServiceType:       "hotspot",
			ProfileName:       profileName,
			SuggestedName:     buildImportPlanName(router.Name, profileName, "hotspot"),
			Conflict:          existing != nil,
			RecommendedAction: "create_network_plan",
		}
		if existing != nil {
			candidate.ExistingNetworkPlanID = &existing.ID
			candidate.ExistingNetworkPlan = existing.Name
			candidate.RecommendedAction = "skip_existing"
		}
		items = append(items, candidate)
	}
	return items
}

func buildPPPoEAccountCandidates(
	router *models.Router,
	existingAccounts []models.ServiceAccount,
	planLookup map[string]RouterImportPlanCandidate,
	secrets []map[string]string,
) []RouterImportAccountPreview {
	items := make([]RouterImportAccountPreview, 0, len(secrets))
	for _, secret := range secrets {
		username := strings.TrimSpace(secret["name"])
		if username == "" {
			continue
		}
		profileName := strings.TrimSpace(secret["profile"])
		existing := matchExistingServiceAccount(existingAccounts, router.ID, "pppoe", username, strings.TrimSpace(secret[".id"]))
		candidate := RouterImportAccountPreview{
			ServiceType:       "pppoe",
			Username:          username,
			RemoteID:          strings.TrimSpace(secret[".id"]),
			ProfileName:       profileName,
			RemoteStatus:      deriveRemoteStatus(secret["disabled"]),
			Comment:           strings.TrimSpace(secret["comment"]),
			Conflict:          existing != nil,
			RecommendedAction: "review_and_link_subscription",
		}
		if existing != nil {
			candidate.ExistingServiceAccountID = &existing.ID
			candidate.ExistingServiceAccount = existing.Username
			candidate.RecommendedAction = "skip_existing"
		}
		attachMatchedPlan(&candidate, planLookup, "pppoe", profileName)
		items = append(items, candidate)
	}
	return items
}

func buildHotspotAccountCandidates(
	router *models.Router,
	existingAccounts []models.ServiceAccount,
	planLookup map[string]RouterImportPlanCandidate,
	users []map[string]string,
) []RouterImportAccountPreview {
	items := make([]RouterImportAccountPreview, 0, len(users))
	for _, user := range users {
		username := strings.TrimSpace(user["name"])
		if username == "" {
			continue
		}
		profileName := strings.TrimSpace(user["profile"])
		existing := matchExistingServiceAccount(existingAccounts, router.ID, "hotspot", username, strings.TrimSpace(user[".id"]))
		candidate := RouterImportAccountPreview{
			ServiceType:       "hotspot",
			Username:          username,
			RemoteID:          strings.TrimSpace(user[".id"]),
			ProfileName:       profileName,
			RemoteStatus:      deriveRemoteStatus(user["disabled"]),
			Comment:           strings.TrimSpace(user["comment"]),
			Conflict:          existing != nil,
			RecommendedAction: "review_and_link_subscription",
		}
		if existing != nil {
			candidate.ExistingServiceAccountID = &existing.ID
			candidate.ExistingServiceAccount = existing.Username
			candidate.RecommendedAction = "skip_existing"
		}
		attachMatchedPlan(&candidate, planLookup, "hotspot", profileName)
		items = append(items, candidate)
	}
	return items
}

func attachMatchedPlan(candidate *RouterImportAccountPreview, planLookup map[string]RouterImportPlanCandidate, serviceType string, profileName string) {
	if candidate == nil {
		return
	}
	item, ok := planLookup[planCandidateKey(serviceType, profileName)]
	if !ok {
		return
	}
	if item.ExistingNetworkPlanID != nil {
		candidate.MatchedNetworkPlanID = item.ExistingNetworkPlanID
		candidate.MatchedNetworkPlan = item.ExistingNetworkPlan
		return
	}
	candidate.MatchedNetworkPlan = item.SuggestedName
}

func matchExistingNetworkPlan(
	items []models.NetworkPlan,
	routerID uuid.UUID,
	serviceType string,
	profileName string,
) *models.NetworkPlan {
	for i := range items {
		item := items[i]
		if item.RouterID == nil || *item.RouterID != routerID {
			continue
		}
		if strings.TrimSpace(strings.ToLower(item.ServiceType)) != strings.TrimSpace(strings.ToLower(serviceType)) {
			continue
		}
		if strings.TrimSpace(strings.ToLower(item.MikrotikProfileName)) != strings.TrimSpace(strings.ToLower(profileName)) {
			continue
		}
		return &item
	}
	return nil
}

func matchExistingServiceAccount(
	items []models.ServiceAccount,
	routerID uuid.UUID,
	serviceType string,
	username string,
	remoteID string,
) *models.ServiceAccount {
	for i := range items {
		item := items[i]
		if strings.TrimSpace(strings.ToLower(item.ServiceType)) != strings.TrimSpace(strings.ToLower(serviceType)) {
			continue
		}
		accountRouterID := resolveServiceAccountRouterID(&item)
		if accountRouterID == nil || *accountRouterID != routerID {
			continue
		}
		if strings.TrimSpace(strings.ToLower(item.Username)) == strings.TrimSpace(strings.ToLower(username)) {
			return &item
		}
		if remoteID != "" && strings.TrimSpace(item.RemoteID) == remoteID {
			return &item
		}
	}
	return nil
}

func resolveServiceAccountRouterID(account *models.ServiceAccount) *uuid.UUID {
	if account == nil {
		return nil
	}
	if account.RouterID != nil {
		return account.RouterID
	}
	if account.NetworkPlan != nil && account.NetworkPlan.RouterID != nil {
		return account.NetworkPlan.RouterID
	}
	return nil
}

func planCandidateKey(serviceType string, profileName string) string {
	return strings.TrimSpace(strings.ToLower(serviceType)) + "::" + strings.TrimSpace(strings.ToLower(profileName))
}

func buildImportPlanName(routerName string, profileName string, serviceType string) string {
	base := strings.TrimSpace(profileName)
	if base == "" {
		base = "Imported"
	}
	return strings.TrimSpace(fmt.Sprintf("%s %s %s", routerName, serviceType, base))
}

func deriveRemoteStatus(disabled string) string {
	value := strings.TrimSpace(strings.ToLower(disabled))
	if value == "true" || value == "yes" {
		return "disabled"
	}
	return "enabled"
}

func resolveRouterName(router *models.Router) string {
	if router == nil {
		return ""
	}
	return router.Name
}

func resolveNetworkPlanName(plan *models.NetworkPlan) string {
	if plan == nil {
		return ""
	}
	return plan.Name
}

func resolveServiceAccountName(account *models.ServiceAccount) string {
	if account == nil {
		return ""
	}
	return account.Username
}

func buildPlanImportNote(item RouterImportPlanCandidate) string {
	if strings.TrimSpace(item.AddressPool) != "" {
		return "Address pool: " + strings.TrimSpace(item.AddressPool)
	}
	return ""
}

func routerSyncEnabled() bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv("FEATURE_ROUTER_SYNC_ENABLED")))
	return value == "1" || value == "true" || value == "yes"
}

func resolveRouterHealthCheckInterval() time.Duration {
	value := strings.TrimSpace(os.Getenv("ROUTER_HEALTHCHECK_INTERVAL"))
	if value == "" {
		return 10 * time.Minute
	}
	duration, err := time.ParseDuration(value)
	if err != nil || duration <= 0 {
		return 10 * time.Minute
	}
	return duration
}

func (s *routerService) markRouterError(router *models.Router, status string, message string) {
	if router == nil {
		return
	}
	router.Status = status
	router.LastError = message
	_ = s.repo.Update(router)
}

func (s *routerService) log(routerID uuid.UUID, level string, action string, message string) {
	if s.logRepo == nil {
		return
	}
	var routerIDPtr *uuid.UUID
	if routerID != uuid.Nil {
		id := routerID
		routerIDPtr = &id
	}
	_ = s.logRepo.Create(&models.ProvisioningLog{
		RouterID: routerIDPtr,
		Level:    level,
		Action:   action,
		Message:  message,
	})
}
