package observability

import (
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type AuthorizationMetric struct {
	Method     string `json:"method"`
	Route      string `json:"route"`
	Permission string `json:"permission"`
	Status     int    `json:"status"`
	Count      uint64 `json:"count"`
}

type SecurityAlert struct {
	Type      string    `json:"type"`
	Route     string    `json:"route,omitempty"`
	Count     int       `json:"count,omitempty"`
	ActorID   uuid.UUID `json:"actor_id,omitempty"`
	TargetID  uuid.UUID `json:"target_id,omitempty"`
	Action    string    `json:"action,omitempty"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
}

type AccessControlSnapshot struct {
	Metrics []AuthorizationMetric `json:"metrics"`
	Alerts  []SecurityAlert       `json:"alerts"`
}

type authorizationMetricKey struct {
	method, route, permission string
	status                    int
}

type AccessControlObserver struct {
	mu             sync.Mutex
	metrics        map[authorizationMetricKey]uint64
	forbiddenTimes map[string][]time.Time
	lastAlertAt    map[string]time.Time
	alerts         []SecurityAlert
	now            func() time.Time
}

var DefaultAccessControl = NewAccessControlObserver()

func NewAccessControlObserver() *AccessControlObserver {
	return &AccessControlObserver{
		metrics:        make(map[authorizationMetricKey]uint64),
		forbiddenTimes: make(map[string][]time.Time),
		lastAlertAt:    make(map[string]time.Time),
		now:            time.Now,
	}
}

func (o *AccessControlObserver) RecordAuthorization(method, route, permission string, status int) {
	if o == nil || (status != 401 && status != 403) {
		return
	}
	now := o.now()
	method = strings.ToUpper(strings.TrimSpace(method))
	route = normalizedLabel(route, "__unknown_route__")
	permission = normalizedLabel(permission, "__authentication__")

	o.mu.Lock()
	defer o.mu.Unlock()
	o.metrics[authorizationMetricKey{method: method, route: route, permission: permission, status: status}]++
	if status != 403 {
		return
	}

	window := forbiddenAlertWindow()
	threshold := forbiddenAlertThreshold()
	cutoff := now.Add(-window)
	times := append(o.forbiddenTimes[route], now)
	firstValid := 0
	for firstValid < len(times) && times[firstValid].Before(cutoff) {
		firstValid++
	}
	times = append([]time.Time(nil), times[firstValid:]...)
	o.forbiddenTimes[route] = times
	if len(times) < threshold || now.Sub(o.lastAlertAt[route]) < window {
		return
	}
	o.lastAlertAt[route] = now
	o.appendAlertLocked(SecurityAlert{
		Type: "forbidden_spike", Route: route, Count: len(times),
		Message: "forbidden access threshold exceeded", CreatedAt: now,
	})
}

func (o *AccessControlObserver) RecordOwnerChange(actorID, targetID uuid.UUID, action string) {
	if o == nil {
		return
	}
	alert := SecurityAlert{
		Type: "owner_change", ActorID: actorID, TargetID: targetID,
		Action: strings.TrimSpace(action), Message: "owner membership changed", CreatedAt: o.now(),
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	o.appendAlertLocked(alert)
}

func (o *AccessControlObserver) Snapshot() AccessControlSnapshot {
	if o == nil {
		return AccessControlSnapshot{}
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	metrics := make([]AuthorizationMetric, 0, len(o.metrics))
	for key, count := range o.metrics {
		metrics = append(metrics, AuthorizationMetric{
			Method: key.method, Route: key.route, Permission: key.permission, Status: key.status, Count: count,
		})
	}
	sort.Slice(metrics, func(i, j int) bool {
		left := metrics[i]
		right := metrics[j]
		if left.Route != right.Route {
			return left.Route < right.Route
		}
		if left.Permission != right.Permission {
			return left.Permission < right.Permission
		}
		if left.Status != right.Status {
			return left.Status < right.Status
		}
		return left.Method < right.Method
	})
	alerts := append([]SecurityAlert(nil), o.alerts...)
	return AccessControlSnapshot{Metrics: metrics, Alerts: alerts}
}

func (o *AccessControlObserver) Reset() {
	if o == nil {
		return
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	o.metrics = make(map[authorizationMetricKey]uint64)
	o.forbiddenTimes = make(map[string][]time.Time)
	o.lastAlertAt = make(map[string]time.Time)
	o.alerts = nil
}

func (o *AccessControlObserver) appendAlertLocked(alert SecurityAlert) {
	o.alerts = append(o.alerts, alert)
	if len(o.alerts) > 100 {
		o.alerts = append([]SecurityAlert(nil), o.alerts[len(o.alerts)-100:]...)
	}
	log.Printf(
		"security_alert type=%s route=%s count=%d actor_id=%s target_id=%s action=%s message=%s",
		alert.Type, alert.Route, alert.Count, alert.ActorID, alert.TargetID, alert.Action, alert.Message,
	)
}

func forbiddenAlertThreshold() int {
	value, err := strconv.Atoi(strings.TrimSpace(os.Getenv("AUTHZ_FORBIDDEN_ALERT_THRESHOLD")))
	if err != nil || value < 1 {
		return 20
	}
	return value
}

func forbiddenAlertWindow() time.Duration {
	value, err := time.ParseDuration(strings.TrimSpace(os.Getenv("AUTHZ_FORBIDDEN_ALERT_WINDOW")))
	if err != nil || value <= 0 {
		return 5 * time.Minute
	}
	return value
}

func normalizedLabel(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}
