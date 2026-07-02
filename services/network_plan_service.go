package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Agushim/go_wifi_billing/lib"
	"github.com/Agushim/go_wifi_billing/models"
	"github.com/Agushim/go_wifi_billing/repositories"
	"github.com/google/uuid"
)

type NetworkPlanSyncItem struct {
	ServiceType           string     `json:"service_type"`
	ProfileName           string     `json:"profile_name"`
	AddressPool           string     `json:"address_pool"`
	SuggestedName         string     `json:"suggested_name"`
	DownloadKbps          int        `json:"download_kbps"`
	UploadKbps            int        `json:"upload_kbps"`
	ExistingNetworkPlanID *uuid.UUID `json:"existing_network_plan_id,omitempty"`
	ExistingNetworkPlan   string     `json:"existing_network_plan,omitempty"`
	Action                string     `json:"action"`
}

type NetworkPlanSyncResponse struct {
	RouterID   uuid.UUID             `json:"router_id"`
	RouterName string                `json:"router_name"`
	Mode       string                `json:"mode"`
	Total      int                   `json:"total"`
	Created    int                   `json:"created"`
	Skipped    int                   `json:"skipped"`
	Items      []NetworkPlanSyncItem `json:"items"`
	SyncedAt   time.Time             `json:"synced_at"`
}

type NetworkPlanService interface {
	Create(plan *models.NetworkPlan) (*models.NetworkPlan, error)
	GetAll() ([]models.NetworkPlan, error)
	GetByID(id uuid.UUID) (*models.NetworkPlan, error)
	Update(id uuid.UUID, input *models.NetworkPlan) (*models.NetworkPlan, error)
	Delete(id uuid.UUID) error
	SyncFromRouter(routerID uuid.UUID, mode string) (*NetworkPlanSyncResponse, error)
}

type networkPlanService struct {
	repo       repositories.NetworkPlanRepository
	routerRepo repositories.RouterRepository
}

func NewNetworkPlanService(repo repositories.NetworkPlanRepository, routerRepo repositories.RouterRepository) NetworkPlanService {
	return &networkPlanService{repo: repo, routerRepo: routerRepo}
}

func (s *networkPlanService) Create(plan *models.NetworkPlan) (*models.NetworkPlan, error) {
	if err := validateNetworkPlan(plan); err != nil {
		return nil, err
	}
	if err := s.repo.Create(plan); err != nil {
		return nil, err
	}
	return plan, nil
}

func (s *networkPlanService) GetAll() ([]models.NetworkPlan, error) {
	return s.repo.FindAll()
}

func (s *networkPlanService) GetByID(id uuid.UUID) (*models.NetworkPlan, error) {
	return s.repo.FindByID(id)
}

func (s *networkPlanService) Update(id uuid.UUID, input *models.NetworkPlan) (*models.NetworkPlan, error) {
	if err := validateNetworkPlan(input); err != nil {
		return nil, err
	}
	existing, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	existing.Name = input.Name
	existing.ServiceType = input.ServiceType
	existing.RouterID = input.RouterID
	existing.MikrotikProfileName = input.MikrotikProfileName
	existing.AddressPool = input.AddressPool
	existing.DownloadKbps = input.DownloadKbps
	existing.UploadKbps = input.UploadKbps
	existing.Description = input.Description
	if err := s.repo.Update(existing); err != nil {
		return nil, err
	}
	return existing, nil
}

func (s *networkPlanService) Delete(id uuid.UUID) error {
	return s.repo.Delete(id)
}

func (s *networkPlanService) SyncFromRouter(routerID uuid.UUID, mode string) (*NetworkPlanSyncResponse, error) {
	mode = normalizeImportMode(mode)
	if mode == "" {
		return nil, errors.New("unsupported sync mode")
	}

	router, err := s.routerRepo.FindByID(routerID)
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

	runner, err := lib.NewMikrotikRunner(router.Host, router.Port, router.UseTLS, 5*time.Second, 5*time.Second)
	if err != nil {
		return nil, err
	}
	defer runner.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	login := runner.Login(ctx, router.Username, password)
	if !login.Success {
		return nil, login.Err
	}

	existingPlans, err := s.repo.FindAll()
	if err != nil {
		return nil, err
	}

	items := make([]NetworkPlanSyncItem, 0)
	if mode == "all" || mode == "pppoe" {
		result := runner.RunReadOnly(ctx, 2, "/ppp/profile/print")
		if !result.Success {
			return nil, result.Err
		}
		items = append(items, buildNetworkPlanSyncItems(router, existingPlans, "pppoe", result.Items)...)
	}
	if mode == "all" || mode == "hotspot" {
		result := runner.RunReadOnly(ctx, 2, "/ip/hotspot/user/profile/print")
		if !result.Success {
			return nil, result.Err
		}
		items = append(items, buildNetworkPlanSyncItems(router, existingPlans, "hotspot", result.Items)...)
	}

	created := 0
	skipped := 0
	for i := range items {
		item := &items[i]
		if item.ExistingNetworkPlanID != nil {
			item.Action = "skip_existing"
			skipped++
			continue
		}

		plan := &models.NetworkPlan{
			Name:                item.SuggestedName,
			ServiceType:         item.ServiceType,
			RouterID:            &router.ID,
			MikrotikProfileName: item.ProfileName,
			AddressPool:         item.AddressPool,
			DownloadKbps:        item.DownloadKbps,
			UploadKbps:          item.UploadKbps,
			Description:         fmt.Sprintf("Imported from MikroTik %s profile", item.ServiceType),
		}
		if err := s.repo.Create(plan); err != nil {
			return nil, err
		}
		item.ExistingNetworkPlanID = &plan.ID
		item.ExistingNetworkPlan = plan.Name
		item.Action = "created"
		created++
	}

	now := time.Now()
	router.Status = "connected"
	router.LastSeenAt = &now
	router.LastCheckedAt = &now
	router.LastError = ""
	_ = s.routerRepo.Update(router)

	return &NetworkPlanSyncResponse{
		RouterID:   router.ID,
		RouterName: router.Name,
		Mode:       mode,
		Total:      len(items),
		Created:    created,
		Skipped:    skipped,
		Items:      items,
		SyncedAt:   now,
	}, nil
}

func validateNetworkPlan(plan *models.NetworkPlan) error {
	if plan == nil {
		return errors.New("invalid network plan payload")
	}
	if strings.TrimSpace(plan.Name) == "" {
		return errors.New("network plan name is required")
	}
	if strings.TrimSpace(plan.ServiceType) == "" {
		plan.ServiceType = "pppoe"
	}
	return nil
}

func buildNetworkPlanSyncItems(
	router *models.Router,
	existingPlans []models.NetworkPlan,
	serviceType string,
	profiles []map[string]string,
) []NetworkPlanSyncItem {
	items := make([]NetworkPlanSyncItem, 0, len(profiles))
	for _, profile := range profiles {
		profileName := strings.TrimSpace(profile["name"])
		if profileName == "" {
			continue
		}
		uploadKbps, downloadKbps := parseMikrotikRateLimit(profile["rate-limit"])
		existing := matchExistingNetworkPlan(existingPlans, router.ID, serviceType, profileName)
		item := NetworkPlanSyncItem{
			ServiceType:   serviceType,
			ProfileName:   profileName,
			AddressPool:   strings.TrimSpace(profile["remote-address"]),
			SuggestedName: buildImportPlanName(router.Name, profileName, serviceType),
			DownloadKbps:  downloadKbps,
			UploadKbps:    uploadKbps,
			Action:        "create",
		}
		if item.AddressPool == "" {
			item.AddressPool = strings.TrimSpace(profile["address-pool"])
		}
		if existing != nil {
			item.ExistingNetworkPlanID = &existing.ID
			item.ExistingNetworkPlan = existing.Name
			item.Action = "skip_existing"
		}
		items = append(items, item)
	}
	return items
}

func parseMikrotikRateLimit(value string) (int, int) {
	fields := strings.Fields(strings.TrimSpace(value))
	if len(fields) == 0 {
		return 0, 0
	}
	parts := strings.Split(fields[0], "/")
	if len(parts) == 1 {
		kbps := parseMikrotikRateKbps(parts[0])
		return kbps, kbps
	}
	return parseMikrotikRateKbps(parts[0]), parseMikrotikRateKbps(parts[1])
}

func parseMikrotikRateKbps(value string) int {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" || value == "0" {
		return 0
	}

	multiplier := 1
	switch {
	case strings.HasSuffix(value, "k"):
		value = strings.TrimSuffix(value, "k")
	case strings.HasSuffix(value, "m"):
		value = strings.TrimSuffix(value, "m")
		multiplier = 1000
	case strings.HasSuffix(value, "g"):
		value = strings.TrimSuffix(value, "g")
		multiplier = 1000 * 1000
	}

	var parsed float64
	if _, err := fmt.Sscanf(value, "%f", &parsed); err != nil {
		return 0
	}
	return int(parsed * float64(multiplier))
}
