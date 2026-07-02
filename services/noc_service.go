package services

import (
	"context"
	"errors"
	"fmt"
	"log"
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

type NOCRouterOverview struct {
	RouterID        uuid.UUID  `json:"router_id"`
	RouterName      string     `json:"router_name"`
	Status          string     `json:"status"`
	Identity        string     `json:"identity"`
	RouterOSVersion string     `json:"routeros_version"`
	BoardName       string     `json:"board_name"`
	CPULoad         int        `json:"cpu_load"`
	FreeMemory      int64      `json:"free_memory"`
	TotalMemory     int64      `json:"total_memory"`
	Uptime          string     `json:"uptime"`
	ActivePPPoE     int        `json:"active_pppoe"`
	ActiveHotspot   int        `json:"active_hotspot"`
	InterfaceCount  int        `json:"interface_count"`
	LastLatencyMS   int64      `json:"last_latency_ms"`
	LastSeenAt      *time.Time `json:"last_seen_at"`
	LastCheckedAt   *time.Time `json:"last_checked_at"`
	LastError       string     `json:"last_error"`
	CollectedAt     time.Time  `json:"collected_at"`
}

type NOCOverview struct {
	TotalRouters        int                 `json:"total_routers"`
	OnlineRouters       int                 `json:"online_routers"`
	OfflineRouters      int                 `json:"offline_routers"`
	TotalOnlineSessions int64               `json:"total_online_sessions"`
	OnlinePPPoE         int64               `json:"online_pppoe"`
	OnlineHotspot       int64               `json:"online_hotspot"`
	InterfaceIssues     []NOCInterfaceIssue `json:"interface_issues"`
	Routers             []NOCRouterOverview `json:"routers"`
	GeneratedAt         time.Time           `json:"generated_at"`
}

type NOCInterfaceIssue struct {
	RouterID    uuid.UUID `json:"router_id"`
	RouterName  string    `json:"router_name"`
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	Running     bool      `json:"running"`
	Disabled    bool      `json:"disabled"`
	CollectedAt time.Time `json:"collected_at"`
}

type NOCCollectionSummary struct {
	TotalRouters int       `json:"total_routers"`
	Succeeded    int       `json:"succeeded"`
	Failed       int       `json:"failed"`
	CollectedAt  time.Time `json:"collected_at"`
	DurationMS   int64     `json:"duration_ms"`
	Errors       []string  `json:"errors,omitempty"`
}

type NOCMetrics struct {
	LastCollectorStartedAt    *time.Time `json:"last_collector_started_at,omitempty"`
	LastCollectorFinishedAt   *time.Time `json:"last_collector_finished_at,omitempty"`
	LastCollectorDurationMS   int64      `json:"last_collector_duration_ms"`
	LastCollectorSucceeded    int        `json:"last_collector_succeeded"`
	LastCollectorFailed       int        `json:"last_collector_failed"`
	LastCollectorTotalRouters int        `json:"last_collector_total_routers"`
	CollectorSuccessTotal     int64      `json:"collector_success_total"`
	CollectorFailureTotal     int64      `json:"collector_failure_total"`
	MikrotikCommandCount      int64      `json:"mikrotik_command_count"`
	MikrotikCommandLatencyMS  int64      `json:"mikrotik_command_latency_ms"`
	ProvisioningQueueSize     int64      `json:"provisioning_queue_size"`
	RetentionLastAppliedAt    *time.Time `json:"retention_last_applied_at,omitempty"`
	LastRetentionError        string     `json:"last_retention_error,omitempty"`
}

type NOCCustomerRow struct {
	ServiceAccountID  uuid.UUID  `json:"service_account_id"`
	CustomerName      string     `json:"customer_name"`
	CustomerPhone     string     `json:"customer_phone"`
	CoverageName      string     `json:"coverage_name"`
	PackageName       string     `json:"package_name"`
	RouterID          *uuid.UUID `json:"router_id"`
	RouterName        string     `json:"router_name"`
	ServiceType       string     `json:"service_type"`
	Username          string     `json:"username"`
	BillingStatus     string     `json:"billing_status"`
	OperationalStatus string     `json:"operational_status"`
	ProfileName       string     `json:"profile_name"`
	LastIPAddress     string     `json:"last_ip_address"`
	LastCallerID      string     `json:"last_caller_id"`
	LastSessionUptime string     `json:"last_session_uptime"`
	LastRXBytes       int64      `json:"last_rx_bytes"`
	LastTXBytes       int64      `json:"last_tx_bytes"`
	LastOnlineAt      *time.Time `json:"last_online_at"`
	LastOfflineAt     *time.Time `json:"last_offline_at"`
	LastSyncedAt      *time.Time `json:"last_synced_at"`
}

type NOCService interface {
	CollectAll() (*NOCCollectionSummary, error)
	GetOverview() (*NOCOverview, error)
	GetRouters() ([]NOCRouterOverview, error)
	GetRouterSnapshots(id uuid.UUID, limit int) ([]models.RouterSnapshot, error)
	GetRouterInterfaces(id uuid.UUID, limit int) ([]models.RouterInterfaceSnapshot, error)
	GetCustomers(status string, routerID string, limit int) ([]NOCCustomerRow, error)
	GetReconciliationFindings(status string) ([]models.ReconciliationFinding, error)
	ResolveReconciliationFinding(id uuid.UUID) error
	GetMetrics() (*NOCMetrics, error)
	StartCollectorScheduler()
}

type nocService struct {
	routerRepo         repositories.RouterRepository
	nocRepo            repositories.NOCRepository
	serviceAccountRepo repositories.ServiceAccountRepository
	logRepo            repositories.ProvisioningLogRepository
	jobRepo            repositories.ProvisioningJobRepository
	metricsMu          sync.Mutex
	metrics            NOCMetrics
}

func NewNOCService(
	routerRepo repositories.RouterRepository,
	nocRepo repositories.NOCRepository,
	serviceAccountRepo repositories.ServiceAccountRepository,
	logRepo repositories.ProvisioningLogRepository,
	jobRepo repositories.ProvisioningJobRepository,
) NOCService {
	return &nocService{
		routerRepo:         routerRepo,
		nocRepo:            nocRepo,
		serviceAccountRepo: serviceAccountRepo,
		logRepo:            logRepo,
		jobRepo:            jobRepo,
	}
}

func (s *nocService) CollectAll() (*NOCCollectionSummary, error) {
	startedAt := time.Now()
	s.metricsMu.Lock()
	s.metrics.LastCollectorStartedAt = &startedAt
	s.metricsMu.Unlock()

	routers, err := s.routerRepo.FindAll()
	if err != nil {
		return nil, err
	}

	accounts, err := s.serviceAccountRepo.FindAll()
	if err != nil {
		return nil, err
	}

	summary := &NOCCollectionSummary{
		TotalRouters: len(routers),
		CollectedAt:  time.Now(),
	}

	type collectResult struct {
		routerName string
		err        error
	}
	workerCount := resolveNOCCollectorWorkers()
	if workerCount > len(routers) && len(routers) > 0 {
		workerCount = len(routers)
	}
	jobs := make(chan models.Router)
	results := make(chan collectResult, len(routers))

	var wg sync.WaitGroup
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for router := range jobs {
				routerCopy := router
				results <- collectResult{
					routerName: routerCopy.Name,
					err:        s.collectRouter(&routerCopy, accounts),
				}
			}
		}()
	}
	for _, router := range routers {
		jobs <- router
	}
	close(jobs)
	wg.Wait()
	close(results)

	for result := range results {
		if result.err != nil {
			summary.Failed++
			summary.Errors = append(summary.Errors, fmt.Sprintf("%s: %v", result.routerName, result.err))
			continue
		}
		summary.Succeeded++
	}

	finishedAt := time.Now()
	summary.DurationMS = finishedAt.Sub(startedAt).Milliseconds()
	s.afterCollection(summary, startedAt, finishedAt)

	return summary, nil
}

func (s *nocService) collectRouter(router *models.Router, accounts []models.ServiceAccount) error {
	if router == nil {
		return errors.New("router is nil")
	}
	if nocRequireRouterTLS() && !router.UseTLS {
		return errors.New("router TLS is required by NOC policy")
	}

	password, err := lib.DecryptSecret(router.PasswordEncrypted)
	if err != nil {
		return err
	}
	if strings.TrimSpace(password) == "" {
		return errors.New("router password is not configured")
	}

	start := time.Now()
	runner, err := lib.NewMikrotikRunner(router.Host, router.Port, router.UseTLS, 5*time.Second, 5*time.Second)
	if err != nil {
		s.markRouterError(router, lib.ClassifyMikrotikError(err), err.Error())
		return err
	}
	defer runner.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	login := runner.Login(ctx, router.Username, password)
	s.logRouterCommand(router.ID, "noc_collect", login)
	if !login.Success {
		s.markRouterError(router, login.ErrorCode, login.Err.Error())
		return login.Err
	}

	identity := s.runReadOnly(ctx, runner, router.ID, "/system/identity/print")
	resource := s.runReadOnly(ctx, runner, router.ID, "/system/resource/print")
	interfaces := s.runReadOnly(ctx, runner, router.ID, "/interface/print")
	pppActive := s.runReadOnly(ctx, runner, router.ID, "/ppp/active/print")
	hotspotActive := s.runReadOnly(ctx, runner, router.ID, "/ip/hotspot/active/print")
	routerLogs := s.runReadOnly(ctx, runner, router.ID, "/log/print")
	pppSecrets := s.runReadOnly(ctx, runner, router.ID, "/ppp/secret/print")
	hotspotUsers := s.runReadOnly(ctx, runner, router.ID, "/ip/hotspot/user/print")

	results := []*lib.MikrotikCommandResult{identity, resource, interfaces, pppActive, hotspotActive, routerLogs, pppSecrets, hotspotUsers}
	for _, result := range results {
		if result != nil && !result.Success {
			s.markRouterError(router, result.ErrorCode, result.Err.Error())
			return result.Err
		}
	}

	now := time.Now()
	identityItem := firstItem(identity)
	resourceItem := firstItem(resource)

	snapshot := &models.RouterSnapshot{
		RouterID:        router.ID,
		Identity:        strings.TrimSpace(identityItem["name"]),
		RouterOSVersion: strings.TrimSpace(resourceItem["version"]),
		BoardName:       strings.TrimSpace(resourceItem["board-name"]),
		Architecture:    strings.TrimSpace(resourceItem["architecture-name"]),
		CPULoad:         parseInt(resourceItem["cpu-load"]),
		FreeMemory:      parseInt64(resourceItem["free-memory"]),
		TotalMemory:     parseInt64(resourceItem["total-memory"]),
		FreeHDDSpace:    parseInt64(resourceItem["free-hdd-space"]),
		TotalHDDSpace:   parseInt64(resourceItem["total-hdd-space"]),
		Uptime:          strings.TrimSpace(resourceItem["uptime"]),
		ActivePPPoE:     len(pppActive.Items),
		ActiveHotspot:   len(hotspotActive.Items),
		InterfaceCount:  len(interfaces.Items),
		CollectedAt:     now,
	}
	if err := s.nocRepo.CreateRouterSnapshot(snapshot); err != nil {
		return err
	}

	if err := s.nocRepo.CreateInterfaceSnapshots(buildInterfaceSnapshots(router.ID, interfaces.Items, now)); err != nil {
		return err
	}
	sessionSnapshots := buildSessionSnapshots(router.ID, accounts, pppActive.Items, hotspotActive.Items, now)
	if err := s.nocRepo.CreateServiceSessionSnapshots(sessionSnapshots); err != nil {
		return err
	}
	if err := s.nocRepo.CreateRouterEventLogs(buildRouterEventLogs(router.ID, routerLogs.Items, now)); err != nil {
		return err
	}
	if err := s.reconcileRouterAccounts(router.ID, accounts, sessionSnapshots, pppSecrets.Items, hotspotUsers.Items, now); err != nil {
		return err
	}

	router.Status = "connected"
	router.LastSeenAt = &now
	router.LastCheckedAt = &now
	router.LastError = ""
	router.Identity = snapshot.Identity
	router.RouterOSVersion = snapshot.RouterOSVersion
	router.BoardName = snapshot.BoardName
	router.Architecture = snapshot.Architecture
	router.Uptime = snapshot.Uptime
	router.LastLatencyMS = time.Since(start).Milliseconds()
	_ = s.routerRepo.Update(router)

	return nil
}

func (s *nocService) reconcileRouterAccounts(
	routerID uuid.UUID,
	accounts []models.ServiceAccount,
	sessions []models.ServiceSessionSnapshot,
	pppSecrets []map[string]string,
	hotspotUsers []map[string]string,
	now time.Time,
) error {
	routerAccounts := filterAccountsByRouter(accounts, routerID)
	sessionByAccount := make(map[uuid.UUID]models.ServiceSessionSnapshot)
	for _, session := range sessions {
		if session.ServiceAccountID != nil {
			sessionByAccount[*session.ServiceAccountID] = session
		}
	}

	remoteByKey := make(map[string]remoteAccount)
	for _, secret := range pppSecrets {
		account := remoteAccountFromItem("pppoe", secret)
		if account.Username != "" {
			remoteByKey[remoteAccountKey(account.ServiceType, account.Username)] = account
		}
	}
	for _, user := range hotspotUsers {
		account := remoteAccountFromItem("hotspot", user)
		if account.Username != "" {
			remoteByKey[remoteAccountKey(account.ServiceType, account.Username)] = account
		}
	}

	matchedRemoteKeys := make(map[string]bool)
	for i := range routerAccounts {
		account := routerAccounts[i]
		session, online := sessionByAccount[account.ID]
		remote, remoteExists := remoteByKey[remoteAccountKey(account.ServiceType, account.Username)]
		if remoteExists {
			matchedRemoteKeys[remoteAccountKey(account.ServiceType, account.Username)] = true
		}

		account.LastSyncedAt = &now
		if online {
			account.OperationalStatus = "online"
			account.LastOnlineAt = &now
			account.LastIPAddress = session.Address
			account.LastCallerID = session.CallerID
			account.LastSessionUptime = session.Uptime
			account.LastRXBytes = session.RXBytes
			account.LastTXBytes = session.TXBytes
		} else {
			account.OperationalStatus = "offline"
			account.LastOfflineAt = &now
		}

		if !remoteExists && isBusinessActive(account.Status) {
			account.OperationalStatus = "mismatch"
			if err := s.upsertFinding(routerID, &account.ID, "billing_account_missing_on_router", "high", fmt.Sprintf("Service account %s aktif di billing tetapi tidak ditemukan di MikroTik", account.Username), "review_and_create_or_link_remote_account", remoteAccount{}); err != nil {
				return err
			}
		}

		if remoteExists {
			expectedProfile := expectedAccountProfile(&account)
			if expectedProfile != "" && remote.ProfileName != "" && !strings.EqualFold(expectedProfile, remote.ProfileName) {
				account.OperationalStatus = "mismatch"
				if err := s.upsertFinding(routerID, &account.ID, "profile_mismatch", "medium", fmt.Sprintf("Profile MikroTik %s berbeda dari NetworkPlan %s untuk %s", remote.ProfileName, expectedProfile, account.Username), "apply_plan_profile_to_router", remote); err != nil {
					return err
				}
			}
			if remote.Status == "disabled" && isBusinessActive(account.Status) {
				account.OperationalStatus = "mismatch"
				if err := s.upsertFinding(routerID, &account.ID, "router_disabled_billing_active", "high", fmt.Sprintf("Service account %s aktif di billing tetapi disabled di MikroTik", account.Username), "enable_or_unsuspend_remote_account", remote); err != nil {
					return err
				}
			}
			if remote.Status == "enabled" && isBusinessInactive(account.Status) {
				account.OperationalStatus = "mismatch"
				if err := s.upsertFinding(routerID, &account.ID, "router_enabled_billing_inactive", "high", fmt.Sprintf("Service account %s %s di billing tetapi enabled di MikroTik", account.Username, account.Status), "disable_or_suspend_remote_account", remote); err != nil {
					return err
				}
			}
		}

		if err := s.nocRepo.UpdateServiceAccountOperationalState(&account); err != nil {
			return err
		}
	}

	for key, remote := range remoteByKey {
		if matchedRemoteKeys[key] {
			continue
		}
		if err := s.upsertFinding(routerID, nil, "router_account_missing_in_billing", "medium", fmt.Sprintf("Akun %s %s ada di MikroTik tetapi tidak terhubung ke billing", remote.ServiceType, remote.Username), "review_and_link_subscription", remote); err != nil {
			return err
		}
	}

	return nil
}

func (s *nocService) runReadOnly(ctx context.Context, runner *lib.MikrotikRunner, routerID uuid.UUID, command string) *lib.MikrotikCommandResult {
	result := runner.RunReadOnly(ctx, 2, command)
	s.logRouterCommand(routerID, "noc_collect", result)
	return result
}

func (s *nocService) GetOverview() (*NOCOverview, error) {
	routers, err := s.routerRepo.FindAll()
	if err != nil {
		return nil, err
	}
	snapshots, err := s.nocRepo.GetLatestRouterSnapshots()
	if err != nil {
		return nil, err
	}

	snapshotByRouter := make(map[uuid.UUID]models.RouterSnapshot, len(snapshots))
	for _, snapshot := range snapshots {
		snapshotByRouter[snapshot.RouterID] = snapshot
	}

	now := time.Now()
	since := now.Add(-15 * time.Minute)
	totalOnline, _ := s.nocRepo.CountOnlineSessionsSince(since)
	onlinePPPoE, _ := s.nocRepo.CountOnlineSessionsByTypeSince("pppoe", since)
	onlineHotspot, _ := s.nocRepo.CountOnlineSessionsByTypeSince("hotspot", since)
	interfaceIssues, _ := s.nocRepo.GetRecentInterfaceIssues(since, 10)

	overview := &NOCOverview{
		TotalRouters:        len(routers),
		TotalOnlineSessions: totalOnline,
		OnlinePPPoE:         onlinePPPoE,
		OnlineHotspot:       onlineHotspot,
		GeneratedAt:         now,
		Routers:             make([]NOCRouterOverview, 0, len(routers)),
		InterfaceIssues:     make([]NOCInterfaceIssue, 0, len(interfaceIssues)),
	}

	for _, router := range routers {
		if strings.EqualFold(strings.TrimSpace(router.Status), "connected") {
			overview.OnlineRouters++
		} else {
			overview.OfflineRouters++
		}

		item := NOCRouterOverview{
			RouterID:        router.ID,
			RouterName:      router.Name,
			Status:          router.Status,
			Identity:        router.Identity,
			RouterOSVersion: router.RouterOSVersion,
			BoardName:       router.BoardName,
			Uptime:          router.Uptime,
			LastLatencyMS:   router.LastLatencyMS,
			LastSeenAt:      router.LastSeenAt,
			LastCheckedAt:   router.LastCheckedAt,
			LastError:       router.LastError,
		}
		if snapshot, ok := snapshotByRouter[router.ID]; ok {
			item.Identity = coalesce(item.Identity, snapshot.Identity)
			item.RouterOSVersion = coalesce(item.RouterOSVersion, snapshot.RouterOSVersion)
			item.BoardName = coalesce(item.BoardName, snapshot.BoardName)
			item.CPULoad = snapshot.CPULoad
			item.FreeMemory = snapshot.FreeMemory
			item.TotalMemory = snapshot.TotalMemory
			item.Uptime = coalesce(item.Uptime, snapshot.Uptime)
			item.ActivePPPoE = snapshot.ActivePPPoE
			item.ActiveHotspot = snapshot.ActiveHotspot
			item.InterfaceCount = snapshot.InterfaceCount
			item.CollectedAt = snapshot.CollectedAt
		}
		overview.Routers = append(overview.Routers, item)
	}

	for _, issue := range interfaceIssues {
		routerName := ""
		if issue.Router != nil {
			routerName = issue.Router.Name
		}
		overview.InterfaceIssues = append(overview.InterfaceIssues, NOCInterfaceIssue{
			RouterID:    issue.RouterID,
			RouterName:  routerName,
			Name:        issue.Name,
			Type:        issue.Type,
			Running:     issue.Running,
			Disabled:    issue.Disabled,
			CollectedAt: issue.CollectedAt,
		})
	}

	return overview, nil
}

func (s *nocService) GetCustomers(status string, routerID string, limit int) ([]NOCCustomerRow, error) {
	var parsedRouterID *uuid.UUID
	if strings.TrimSpace(routerID) != "" {
		id, err := uuid.Parse(routerID)
		if err != nil {
			return nil, errors.New("invalid router_id")
		}
		parsedRouterID = &id
	}
	items, err := s.nocRepo.GetNOCServiceAccounts(strings.TrimSpace(status), parsedRouterID, limit)
	if err != nil {
		return nil, err
	}
	rows := make([]NOCCustomerRow, 0, len(items))
	for _, item := range items {
		rows = append(rows, buildNOCCustomerRow(item))
	}
	return rows, nil
}

func (s *nocService) GetRouters() ([]NOCRouterOverview, error) {
	overview, err := s.GetOverview()
	if err != nil {
		return nil, err
	}
	return overview.Routers, nil
}

func (s *nocService) GetRouterSnapshots(id uuid.UUID, limit int) ([]models.RouterSnapshot, error) {
	return s.nocRepo.GetRouterSnapshots(id, limit)
}

func (s *nocService) GetRouterInterfaces(id uuid.UUID, limit int) ([]models.RouterInterfaceSnapshot, error) {
	return s.nocRepo.GetRouterInterfaceSnapshots(id, limit)
}

func (s *nocService) GetReconciliationFindings(status string) ([]models.ReconciliationFinding, error) {
	if strings.TrimSpace(status) == "" {
		status = "open"
	}
	return s.nocRepo.FindReconciliationFindings(status)
}

func (s *nocService) ResolveReconciliationFinding(id uuid.UUID) error {
	return s.nocRepo.ResolveReconciliationFinding(id)
}

func (s *nocService) GetMetrics() (*NOCMetrics, error) {
	s.metricsMu.Lock()
	metrics := s.metrics
	s.metricsMu.Unlock()

	if s.jobRepo != nil {
		pending, err := s.jobRepo.CountByStatus("pending")
		if err != nil {
			return nil, err
		}
		metrics.ProvisioningQueueSize = pending
	}
	return &metrics, nil
}

func (s *nocService) StartCollectorScheduler() {
	if !nocCollectorEnabled() {
		return
	}

	interval := resolveNOCCollectorInterval()
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			if _, err := s.CollectAll(); err != nil {
				log.Printf("[noc] collector failed: %v", err)
			}
		}
	}()
}

func (s *nocService) afterCollection(summary *NOCCollectionSummary, startedAt time.Time, finishedAt time.Time) {
	if summary == nil {
		return
	}
	if _, err := s.nocRepo.AggregateRouterSnapshots("hour", finishedAt.AddDate(0, 0, -7), finishedAt); err != nil {
		log.Printf("[noc] hourly aggregate failed: %v", err)
	}
	if _, err := s.nocRepo.AggregateRouterSnapshots("day", finishedAt.AddDate(0, 0, -90), finishedAt); err != nil {
		log.Printf("[noc] daily aggregate failed: %v", err)
	}

	var retentionError string
	var retentionAppliedAt *time.Time
	if err := s.nocRepo.ApplyRetention(finishedAt); err != nil {
		retentionError = err.Error()
		log.Printf("[noc] retention failed: %v", err)
	} else {
		appliedAt := time.Now()
		retentionAppliedAt = &appliedAt
	}

	s.metricsMu.Lock()
	s.metrics.LastCollectorStartedAt = &startedAt
	s.metrics.LastCollectorFinishedAt = &finishedAt
	s.metrics.LastCollectorDurationMS = summary.DurationMS
	s.metrics.LastCollectorSucceeded = summary.Succeeded
	s.metrics.LastCollectorFailed = summary.Failed
	s.metrics.LastCollectorTotalRouters = summary.TotalRouters
	s.metrics.CollectorSuccessTotal += int64(summary.Succeeded)
	s.metrics.CollectorFailureTotal += int64(summary.Failed)
	if retentionAppliedAt != nil {
		s.metrics.RetentionLastAppliedAt = retentionAppliedAt
	}
	s.metrics.LastRetentionError = retentionError
	s.metricsMu.Unlock()
}

func (s *nocService) upsertFinding(routerID uuid.UUID, serviceAccountID *uuid.UUID, findingType string, severity string, description string, recommendedAction string, remote remoteAccount) error {
	return s.nocRepo.UpsertOpenReconciliationFinding(&models.ReconciliationFinding{
		RouterID:          routerID,
		ServiceAccountID:  serviceAccountID,
		FindingType:       findingType,
		Severity:          severity,
		Description:       description,
		RecommendedAction: recommendedAction,
		Status:            "open",
		RemoteServiceType: remote.ServiceType,
		RemoteUsername:    remote.Username,
		RemoteID:          remote.RemoteID,
		RemoteProfileName: remote.ProfileName,
		RemoteStatus:      remote.Status,
		DetectedAt:        time.Now(),
	})
}

func (s *nocService) markRouterError(router *models.Router, status string, message string) {
	if router == nil {
		return
	}
	now := time.Now()
	router.Status = status
	router.LastError = message
	router.LastCheckedAt = &now
	_ = s.routerRepo.Update(router)
}

func (s *nocService) logRouterCommand(routerID uuid.UUID, action string, result *lib.MikrotikCommandResult) {
	if result == nil {
		return
	}
	s.observeMikrotikCommand(result.Duration)
	if s.logRepo == nil {
		return
	}
	level := "info"
	status := "succeeded"
	if !result.Success {
		level = "error"
		status = "failed"
	}
	message := fmt.Sprintf(
		"router command %s: command=%s duration_ms=%d error_code=%s",
		status,
		result.Command,
		result.Duration.Milliseconds(),
		result.ErrorCode,
	)
	if result.Err != nil {
		message = fmt.Sprintf("%s error=%s", message, result.Err.Error())
	}

	id := routerID
	_ = s.logRepo.Create(&models.ProvisioningLog{
		RouterID: &id,
		Level:    level,
		Action:   action,
		Message:  message,
	})
}

func (s *nocService) observeMikrotikCommand(duration time.Duration) {
	s.metricsMu.Lock()
	s.metrics.MikrotikCommandCount++
	s.metrics.MikrotikCommandLatencyMS += duration.Milliseconds()
	s.metricsMu.Unlock()
}

func firstItem(result *lib.MikrotikCommandResult) map[string]string {
	if result == nil || len(result.Items) == 0 {
		return map[string]string{}
	}
	return result.Items[0]
}

func buildInterfaceSnapshots(routerID uuid.UUID, items []map[string]string, collectedAt time.Time) []models.RouterInterfaceSnapshot {
	snapshots := make([]models.RouterInterfaceSnapshot, 0, len(items))
	for _, item := range items {
		name := strings.TrimSpace(item["name"])
		if name == "" {
			continue
		}
		snapshots = append(snapshots, models.RouterInterfaceSnapshot{
			RouterID:    routerID,
			Name:        name,
			Type:        strings.TrimSpace(item["type"]),
			Running:     parseRouterBool(item["running"]),
			Disabled:    parseRouterBool(item["disabled"]),
			RXPacket:    parseInt64(item["rx-packet"]),
			TXPacket:    parseInt64(item["tx-packet"]),
			CollectedAt: collectedAt,
		})
	}
	return snapshots
}

func buildSessionSnapshots(routerID uuid.UUID, accounts []models.ServiceAccount, pppActive []map[string]string, hotspotActive []map[string]string, collectedAt time.Time) []models.ServiceSessionSnapshot {
	snapshots := make([]models.ServiceSessionSnapshot, 0, len(pppActive)+len(hotspotActive))
	for _, item := range pppActive {
		username := strings.TrimSpace(item["name"])
		if username == "" {
			continue
		}
		remoteID := strings.TrimSpace(item[".id"])
		snapshots = append(snapshots, models.ServiceSessionSnapshot{
			RouterID:         routerID,
			ServiceAccountID: findServiceAccountID(accounts, routerID, "pppoe", username, remoteID),
			ServiceType:      "pppoe",
			Username:         username,
			RemoteID:         remoteID,
			Address:          strings.TrimSpace(item["address"]),
			Uptime:           strings.TrimSpace(item["uptime"]),
			CallerID:         strings.TrimSpace(item["caller-id"]),
			Online:           true,
			RXBytes:          parseInt64(item["bytes-in"]),
			TXBytes:          parseInt64(item["bytes-out"]),
			CollectedAt:      collectedAt,
		})
	}
	for _, item := range hotspotActive {
		username := strings.TrimSpace(item["user"])
		if username == "" {
			username = strings.TrimSpace(item["name"])
		}
		if username == "" {
			continue
		}
		remoteID := strings.TrimSpace(item[".id"])
		snapshots = append(snapshots, models.ServiceSessionSnapshot{
			RouterID:         routerID,
			ServiceAccountID: findServiceAccountID(accounts, routerID, "hotspot", username, remoteID),
			ServiceType:      "hotspot",
			Username:         username,
			RemoteID:         remoteID,
			Address:          strings.TrimSpace(item["address"]),
			Uptime:           strings.TrimSpace(item["uptime"]),
			CallerID:         strings.TrimSpace(item["mac-address"]),
			Online:           true,
			RXBytes:          parseInt64(item["bytes-in"]),
			TXBytes:          parseInt64(item["bytes-out"]),
			CollectedAt:      collectedAt,
		})
	}
	return snapshots
}

func buildRouterEventLogs(routerID uuid.UUID, items []map[string]string, collectedAt time.Time) []models.RouterEventLog {
	maxItems := 50
	if len(items) < maxItems {
		maxItems = len(items)
	}
	logs := make([]models.RouterEventLog, 0, maxItems)
	for i := 0; i < maxItems; i++ {
		item := items[i]
		message := strings.TrimSpace(item["message"])
		if message == "" {
			continue
		}
		logs = append(logs, models.RouterEventLog{
			RouterID:    routerID,
			Topic:       strings.TrimSpace(item["topics"]),
			Message:     message,
			Severity:    deriveRouterLogSeverity(item["topics"]),
			RemoteTime:  strings.TrimSpace(item["time"]),
			CollectedAt: collectedAt,
		})
	}
	return logs
}

type remoteAccount struct {
	ServiceType string
	Username    string
	RemoteID    string
	ProfileName string
	Status      string
}

func remoteAccountFromItem(serviceType string, item map[string]string) remoteAccount {
	username := strings.TrimSpace(item["name"])
	if serviceType == "hotspot" {
		username = strings.TrimSpace(item["name"])
		if username == "" {
			username = strings.TrimSpace(item["user"])
		}
	}
	status := "enabled"
	if parseRouterBool(item["disabled"]) {
		status = "disabled"
	}
	return remoteAccount{
		ServiceType: serviceType,
		Username:    username,
		RemoteID:    strings.TrimSpace(item[".id"]),
		ProfileName: strings.TrimSpace(item["profile"]),
		Status:      status,
	}
}

func remoteAccountKey(serviceType string, username string) string {
	return strings.TrimSpace(strings.ToLower(serviceType)) + "::" + strings.TrimSpace(strings.ToLower(username))
}

func filterAccountsByRouter(accounts []models.ServiceAccount, routerID uuid.UUID) []models.ServiceAccount {
	items := make([]models.ServiceAccount, 0)
	for i := range accounts {
		account := accounts[i]
		accountRouterID := serviceAccountRouterID(&account)
		if accountRouterID != nil && *accountRouterID == routerID {
			items = append(items, account)
		}
	}
	return items
}

func expectedAccountProfile(account *models.ServiceAccount) string {
	plan := nocEffectiveNetworkPlan(account)
	if plan == nil {
		return ""
	}
	return strings.TrimSpace(plan.MikrotikProfileName)
}

func isBusinessActive(status string) bool {
	value := strings.TrimSpace(strings.ToLower(status))
	return value == "active" || value == "pending"
}

func isBusinessInactive(status string) bool {
	value := strings.TrimSpace(strings.ToLower(status))
	return value == "suspended" || value == "terminated"
}

func buildNOCCustomerRow(account models.ServiceAccount) NOCCustomerRow {
	row := NOCCustomerRow{
		ServiceAccountID:  account.ID,
		RouterID:          serviceAccountRouterID(&account),
		ServiceType:       account.ServiceType,
		Username:          account.Username,
		BillingStatus:     account.Status,
		OperationalStatus: account.OperationalStatus,
		ProfileName:       expectedAccountProfile(&account),
		LastIPAddress:     account.LastIPAddress,
		LastCallerID:      account.LastCallerID,
		LastSessionUptime: account.LastSessionUptime,
		LastRXBytes:       account.LastRXBytes,
		LastTXBytes:       account.LastTXBytes,
		LastOnlineAt:      account.LastOnlineAt,
		LastOfflineAt:     account.LastOfflineAt,
		LastSyncedAt:      account.LastSyncedAt,
	}
	if account.Router != nil {
		row.RouterName = account.Router.Name
	} else if account.NetworkPlan != nil && account.NetworkPlan.Router != nil {
		row.RouterName = account.NetworkPlan.Router.Name
	} else if account.Subscription != nil && account.Subscription.NetworkPlan != nil && account.Subscription.NetworkPlan.Router != nil {
		row.RouterName = account.Subscription.NetworkPlan.Router.Name
	}
	if account.Subscription != nil && account.Subscription.Package != nil {
		row.PackageName = account.Subscription.Package.Name
	}
	if account.Subscription != nil && account.Subscription.Customer != nil {
		row.CustomerPhone = ""
		if account.Subscription.Customer.User != nil {
			row.CustomerName = account.Subscription.Customer.User.Name
			row.CustomerPhone = account.Subscription.Customer.User.Phone
		}
		if account.Subscription.Customer.Coverage != nil {
			row.CoverageName = account.Subscription.Customer.Coverage.Name
		}
	}
	return row
}

func findServiceAccountID(accounts []models.ServiceAccount, routerID uuid.UUID, serviceType string, username string, remoteID string) *uuid.UUID {
	for i := range accounts {
		account := accounts[i]
		accountRouterID := serviceAccountRouterID(&account)
		if accountRouterID == nil || *accountRouterID != routerID {
			continue
		}
		if !strings.EqualFold(strings.TrimSpace(account.ServiceType), serviceType) {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(account.Username), username) || (remoteID != "" && strings.TrimSpace(account.RemoteID) == remoteID) {
			id := account.ID
			return &id
		}
	}
	return nil
}

func serviceAccountRouterID(account *models.ServiceAccount) *uuid.UUID {
	if account == nil {
		return nil
	}
	if account.RouterID != nil {
		return account.RouterID
	}
	if account.NetworkPlan != nil && account.NetworkPlan.RouterID != nil {
		return account.NetworkPlan.RouterID
	}
	if account.Subscription != nil && account.Subscription.NetworkPlan != nil && account.Subscription.NetworkPlan.RouterID != nil {
		return account.Subscription.NetworkPlan.RouterID
	}
	return nil
}

func nocEffectiveNetworkPlan(account *models.ServiceAccount) *models.NetworkPlan {
	if account == nil {
		return nil
	}
	if account.NetworkPlan != nil {
		return account.NetworkPlan
	}
	if account.Subscription != nil && account.Subscription.NetworkPlan != nil {
		return account.Subscription.NetworkPlan
	}
	return nil
}

func parseInt(value string) int {
	parsed, _ := strconv.Atoi(strings.TrimSpace(value))
	return parsed
}

func parseInt64(value string) int64 {
	cleaned := strings.TrimSpace(value)
	cleaned = strings.TrimSuffix(cleaned, "bps")
	cleaned = strings.TrimSpace(cleaned)
	parsed, _ := strconv.ParseInt(cleaned, 10, 64)
	return parsed
}

func parseRouterBool(value string) bool {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "true", "yes", "1":
		return true
	default:
		return false
	}
}

func deriveRouterLogSeverity(topics string) string {
	value := strings.ToLower(topics)
	switch {
	case strings.Contains(value, "error") || strings.Contains(value, "critical"):
		return "error"
	case strings.Contains(value, "warning"):
		return "warning"
	default:
		return "info"
	}
}

func coalesce(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func nocCollectorEnabled() bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv("FEATURE_NOC_COLLECTOR_ENABLED")))
	return value == "1" || value == "true" || value == "yes"
}

func resolveNOCCollectorInterval() time.Duration {
	value := strings.TrimSpace(os.Getenv("NOC_COLLECTOR_INTERVAL"))
	if value == "" {
		return 5 * time.Minute
	}
	duration, err := time.ParseDuration(value)
	if err != nil || duration <= 0 {
		return 5 * time.Minute
	}
	return duration
}

func resolveNOCCollectorWorkers() int {
	value := strings.TrimSpace(os.Getenv("NOC_COLLECTOR_WORKERS"))
	if value == "" {
		return 4
	}
	workers, err := strconv.Atoi(value)
	if err != nil || workers <= 0 {
		return 4
	}
	if workers > 32 {
		return 32
	}
	return workers
}

func nocRequireRouterTLS() bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv("NOC_REQUIRE_ROUTER_TLS")))
	return value == "1" || value == "true" || value == "yes"
}
