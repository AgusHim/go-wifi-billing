package repositories

import (
	"strings"
	"time"

	"github.com/Agushim/go_wifi_billing/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type NOCRepository interface {
	CreateRouterSnapshot(snapshot *models.RouterSnapshot) error
	CreateInterfaceSnapshots(items []models.RouterInterfaceSnapshot) error
	CreateServiceSessionSnapshots(items []models.ServiceSessionSnapshot) error
	CreateRouterEventLogs(items []models.RouterEventLog) error
	UpsertOpenReconciliationFinding(finding *models.ReconciliationFinding) error
	ResolveReconciliationFinding(id uuid.UUID) error
	FindReconciliationFindings(status string) ([]models.ReconciliationFinding, error)
	UpdateServiceAccountOperationalState(account *models.ServiceAccount) error
	GetLatestRouterSnapshots() ([]models.RouterSnapshot, error)
	GetRouterSnapshots(routerID uuid.UUID, limit int) ([]models.RouterSnapshot, error)
	GetRouterInterfaceSnapshots(routerID uuid.UUID, limit int) ([]models.RouterInterfaceSnapshot, error)
	GetNOCServiceAccounts(status string, routerID *uuid.UUID, limit int) ([]models.ServiceAccount, error)
	CountOnlineSessionsSince(since time.Time) (int64, error)
	CountOnlineSessionsByTypeSince(serviceType string, since time.Time) (int64, error)
	GetRecentInterfaceIssues(since time.Time, limit int) ([]models.RouterInterfaceSnapshot, error)
	AggregateRouterSnapshots(granularity string, since time.Time, now time.Time) (int, error)
	ApplyRetention(now time.Time) error
}

type nocRepository struct {
	db *gorm.DB
}

func NewNOCRepository(db *gorm.DB) NOCRepository {
	return &nocRepository{db: db}
}

func (r *nocRepository) CreateRouterSnapshot(snapshot *models.RouterSnapshot) error {
	return r.db.Create(snapshot).Error
}

func (r *nocRepository) CreateInterfaceSnapshots(items []models.RouterInterfaceSnapshot) error {
	if len(items) == 0 {
		return nil
	}
	return r.db.Create(&items).Error
}

func (r *nocRepository) CreateServiceSessionSnapshots(items []models.ServiceSessionSnapshot) error {
	if len(items) == 0 {
		return nil
	}
	return r.db.Create(&items).Error
}

func (r *nocRepository) CreateRouterEventLogs(items []models.RouterEventLog) error {
	if len(items) == 0 {
		return nil
	}
	return r.db.Create(&items).Error
}

func (r *nocRepository) UpsertOpenReconciliationFinding(finding *models.ReconciliationFinding) error {
	if finding == nil {
		return nil
	}
	query := r.db.Where("router_id = ? AND finding_type = ? AND status = ?", finding.RouterID, finding.FindingType, "open")
	if finding.ServiceAccountID != nil {
		query = query.Where("service_account_id = ?", *finding.ServiceAccountID)
	} else {
		query = query.Where("service_account_id IS NULL").
			Where("remote_service_type = ? AND remote_username = ?", finding.RemoteServiceType, finding.RemoteUsername)
	}

	var existing models.ReconciliationFinding
	err := query.First(&existing).Error
	if err == nil {
		existing.Severity = finding.Severity
		existing.Description = finding.Description
		existing.RecommendedAction = finding.RecommendedAction
		existing.RemoteID = finding.RemoteID
		existing.RemoteProfileName = finding.RemoteProfileName
		existing.RemoteStatus = finding.RemoteStatus
		existing.DetectedAt = finding.DetectedAt
		return r.db.Save(&existing).Error
	}
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}
	if strings.TrimSpace(finding.Status) == "" {
		finding.Status = "open"
	}
	return r.db.Create(finding).Error
}

func (r *nocRepository) ResolveReconciliationFinding(id uuid.UUID) error {
	now := time.Now()
	return r.db.Model(&models.ReconciliationFinding{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{"status": "resolved", "resolved_at": &now}).Error
}

func (r *nocRepository) FindReconciliationFindings(status string) ([]models.ReconciliationFinding, error) {
	var items []models.ReconciliationFinding
	query := r.db.
		Preload("Router").
		Preload("ServiceAccount").
		Preload("ServiceAccount.Subscription").
		Preload("ServiceAccount.Subscription.Customer").
		Preload("ServiceAccount.Subscription.Customer.User").
		Order("detected_at DESC")
	if strings.TrimSpace(status) != "" {
		query = query.Where("status = ?", status)
	}
	err := query.Find(&items).Error
	return items, err
}

func (r *nocRepository) UpdateServiceAccountOperationalState(account *models.ServiceAccount) error {
	if account == nil {
		return nil
	}
	return r.db.Model(&models.ServiceAccount{}).
		Where("id = ?", account.ID).
		Updates(map[string]interface{}{
			"operational_status":  account.OperationalStatus,
			"last_synced_at":      account.LastSyncedAt,
			"last_online_at":      account.LastOnlineAt,
			"last_offline_at":     account.LastOfflineAt,
			"last_ip_address":     account.LastIPAddress,
			"last_caller_id":      account.LastCallerID,
			"last_session_uptime": account.LastSessionUptime,
			"last_rx_bytes":       account.LastRXBytes,
			"last_tx_bytes":       account.LastTXBytes,
		}).Error
}

func (r *nocRepository) GetLatestRouterSnapshots() ([]models.RouterSnapshot, error) {
	var snapshots []models.RouterSnapshot
	err := r.db.
		Preload("Router").
		Where(`router_snapshots.collected_at = (
			SELECT MAX(latest.collected_at)
			FROM router_snapshots AS latest
			WHERE latest.router_id = router_snapshots.router_id
			AND latest.deleted_at IS NULL
		)`).
		Order("router_snapshots.collected_at DESC").
		Find(&snapshots).Error
	return snapshots, err
}

func (r *nocRepository) GetRouterSnapshots(routerID uuid.UUID, limit int) ([]models.RouterSnapshot, error) {
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	var snapshots []models.RouterSnapshot
	err := r.db.
		Preload("Router").
		Where("router_id = ?", routerID).
		Order("collected_at DESC").
		Limit(limit).
		Find(&snapshots).Error
	return snapshots, err
}

func (r *nocRepository) GetRouterInterfaceSnapshots(routerID uuid.UUID, limit int) ([]models.RouterInterfaceSnapshot, error) {
	if limit <= 0 || limit > 1000 {
		limit = 200
	}
	var snapshots []models.RouterInterfaceSnapshot
	err := r.db.
		Preload("Router").
		Where("router_id = ?", routerID).
		Order("collected_at DESC, name ASC").
		Limit(limit).
		Find(&snapshots).Error
	return snapshots, err
}

func (r *nocRepository) GetNOCServiceAccounts(status string, routerID *uuid.UUID, limit int) ([]models.ServiceAccount, error) {
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	var items []models.ServiceAccount
	query := r.db.
		Preload("Router").
		Preload("NetworkPlan").
		Preload("NetworkPlan.Router").
		Preload("Subscription").
		Preload("Subscription.Package").
		Preload("Subscription.Customer").
		Preload("Subscription.Customer.User").
		Preload("Subscription.Customer.Coverage").
		Order("updated_at DESC").
		Limit(limit)
	if strings.TrimSpace(status) != "" {
		query = query.Where("operational_status = ?", status)
	}
	if routerID != nil {
		query = query.Where("router_id = ? OR network_plan_id IN (SELECT id FROM network_plans WHERE router_id = ?)", *routerID, *routerID)
	}
	err := query.Find(&items).Error
	return items, err
}

func (r *nocRepository) CountOnlineSessionsSince(since time.Time) (int64, error) {
	var total int64
	err := r.db.Model(&models.ServiceSessionSnapshot{}).
		Where("online = ? AND collected_at >= ? AND collected_at = (?)", true, since,
			r.db.Model(&models.ServiceSessionSnapshot{}).
				Select("MAX(latest.collected_at)").
				Table("service_session_snapshots AS latest").
				Where("latest.router_id = service_session_snapshots.router_id").
				Where("latest.deleted_at IS NULL"),
		).
		Distinct("router_id", "service_type", "username").
		Count(&total).Error
	return total, err
}

func (r *nocRepository) CountOnlineSessionsByTypeSince(serviceType string, since time.Time) (int64, error) {
	var total int64
	err := r.db.Model(&models.ServiceSessionSnapshot{}).
		Where("online = ? AND LOWER(service_type) = LOWER(?) AND collected_at >= ? AND collected_at = (?)", true, serviceType, since,
			r.db.Model(&models.ServiceSessionSnapshot{}).
				Select("MAX(latest.collected_at)").
				Table("service_session_snapshots AS latest").
				Where("latest.router_id = service_session_snapshots.router_id").
				Where("latest.deleted_at IS NULL"),
		).
		Distinct("router_id", "service_type", "username").
		Count(&total).Error
	return total, err
}

func (r *nocRepository) GetRecentInterfaceIssues(since time.Time, limit int) ([]models.RouterInterfaceSnapshot, error) {
	if limit <= 0 {
		limit = 10
	}
	var items []models.RouterInterfaceSnapshot
	err := r.db.
		Preload("Router").
		Where("collected_at >= ? AND (running = ? OR disabled = ?)", since, false, true).
		Order("collected_at DESC").
		Limit(limit).
		Find(&items).Error
	return items, err
}

func (r *nocRepository) AggregateRouterSnapshots(granularity string, since time.Time, now time.Time) (int, error) {
	granularity = strings.ToLower(strings.TrimSpace(granularity))
	if granularity != "hour" && granularity != "day" {
		granularity = "hour"
	}
	var snapshots []models.RouterSnapshot
	if err := r.db.
		Where("collected_at >= ? AND collected_at <= ?", since, now).
		Order("router_id ASC, collected_at ASC").
		Find(&snapshots).Error; err != nil {
		return 0, err
	}

	type bucketStats struct {
		routerID           uuid.UUID
		bucketAt           time.Time
		cpuTotal           int
		maxCPU             int
		freeMemoryTotal    int64
		activePPPoETotal   int
		activeHotspotTotal int
		maxActiveSessions  int
		sampleCount        int
		firstSampleAt      time.Time
		lastSampleAt       time.Time
	}
	buckets := map[string]*bucketStats{}
	for _, snapshot := range snapshots {
		bucketAt := snapshot.CollectedAt.Truncate(time.Hour)
		if granularity == "day" {
			y, m, d := snapshot.CollectedAt.Date()
			bucketAt = time.Date(y, m, d, 0, 0, 0, 0, snapshot.CollectedAt.Location())
		}
		key := snapshot.RouterID.String() + ":" + granularity + ":" + bucketAt.UTC().Format(time.RFC3339)
		stats := buckets[key]
		if stats == nil {
			stats = &bucketStats{
				routerID:      snapshot.RouterID,
				bucketAt:      bucketAt,
				firstSampleAt: snapshot.CollectedAt,
				lastSampleAt:  snapshot.CollectedAt,
			}
			buckets[key] = stats
		}
		stats.cpuTotal += snapshot.CPULoad
		if snapshot.CPULoad > stats.maxCPU {
			stats.maxCPU = snapshot.CPULoad
		}
		stats.freeMemoryTotal += snapshot.FreeMemory
		stats.activePPPoETotal += snapshot.ActivePPPoE
		stats.activeHotspotTotal += snapshot.ActiveHotspot
		activeSessions := snapshot.ActivePPPoE + snapshot.ActiveHotspot
		if activeSessions > stats.maxActiveSessions {
			stats.maxActiveSessions = activeSessions
		}
		stats.sampleCount++
		if snapshot.CollectedAt.Before(stats.firstSampleAt) {
			stats.firstSampleAt = snapshot.CollectedAt
		}
		if snapshot.CollectedAt.After(stats.lastSampleAt) {
			stats.lastSampleAt = snapshot.CollectedAt
		}
	}

	aggregates := make([]models.RouterMetricAggregate, 0, len(buckets))
	for _, stats := range buckets {
		if stats.sampleCount == 0 {
			continue
		}
		aggregates = append(aggregates, models.RouterMetricAggregate{
			RouterID:          stats.routerID,
			Granularity:       granularity,
			BucketAt:          stats.bucketAt,
			AvgCPULoad:        float64(stats.cpuTotal) / float64(stats.sampleCount),
			MaxCPULoad:        stats.maxCPU,
			AvgFreeMemory:     float64(stats.freeMemoryTotal) / float64(stats.sampleCount),
			AvgActivePPPoE:    float64(stats.activePPPoETotal) / float64(stats.sampleCount),
			AvgActiveHotspot:  float64(stats.activeHotspotTotal) / float64(stats.sampleCount),
			MaxActiveSessions: stats.maxActiveSessions,
			SampleCount:       stats.sampleCount,
			FirstSampleAt:     stats.firstSampleAt,
			LastSampleAt:      stats.lastSampleAt,
		})
	}
	if len(aggregates) == 0 {
		return 0, nil
	}
	err := r.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "router_id"},
			{Name: "granularity"},
			{Name: "bucket_at"},
		},
		DoUpdates: clause.AssignmentColumns([]string{
			"avg_cpu_load",
			"max_cpu_load",
			"avg_free_memory",
			"avg_active_pppoe",
			"avg_active_hotspot",
			"max_active_sessions",
			"sample_count",
			"first_sample_at",
			"last_sample_at",
			"updated_at",
		}),
	}).Create(&aggregates).Error
	return len(aggregates), err
}

func (r *nocRepository) ApplyRetention(now time.Time) error {
	rawCutoff := now.AddDate(0, 0, -7)
	if err := r.db.Where("collected_at < ?", rawCutoff).Delete(&models.RouterSnapshot{}).Error; err != nil {
		return err
	}
	if err := r.db.Where("collected_at < ?", rawCutoff).Delete(&models.RouterInterfaceSnapshot{}).Error; err != nil {
		return err
	}
	if err := r.db.Where("collected_at < ?", rawCutoff).Delete(&models.ServiceSessionSnapshot{}).Error; err != nil {
		return err
	}
	if err := r.db.Where("collected_at < ?", rawCutoff).Delete(&models.RouterEventLog{}).Error; err != nil {
		return err
	}
	if err := r.db.Where("granularity = ? AND bucket_at < ?", "hour", now.AddDate(0, 0, -90)).
		Delete(&models.RouterMetricAggregate{}).Error; err != nil {
		return err
	}
	return r.db.Where("granularity = ? AND bucket_at < ?", "day", now.AddDate(-1, 0, 0)).
		Delete(&models.RouterMetricAggregate{}).Error
}

func FindServiceAccountIDForSession(accounts []models.ServiceAccount, routerID uuid.UUID, serviceType string, username string, remoteID string) *uuid.UUID {
	for i := range accounts {
		account := accounts[i]
		if account.RouterID != nil && *account.RouterID != routerID {
			continue
		}
		if account.NetworkPlan != nil && account.NetworkPlan.RouterID != nil && *account.NetworkPlan.RouterID != routerID {
			continue
		}
		if account.RouterID == nil && (account.NetworkPlan == nil || account.NetworkPlan.RouterID == nil) {
			continue
		}
		if !equalFoldTrim(account.ServiceType, serviceType) {
			continue
		}
		if equalFoldTrim(account.Username, username) || (remoteID != "" && account.RemoteID == remoteID) {
			id := account.ID
			return &id
		}
	}
	return nil
}

func equalFoldTrim(a string, b string) bool {
	return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
}
