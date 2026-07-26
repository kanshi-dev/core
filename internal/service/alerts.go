package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/kanshi-dev/core/internal/db"
)

var (
	ErrInvalidRule  = errors.New("invalid alert rule")
	ErrRuleNotFound = errors.New("alert rule not found")
)

// AlertMetrics is the set of metrics a rule may target. agent.offline has no
// metric row; it is derived from agents.last_seen with the threshold in seconds.
var AlertMetrics = map[string]bool{
	"cpu.used_percent":  true,
	"mem.used_percent":  true,
	"disk.used_percent": true,
	"agent.offline":     true,
}

// AlertRuleInput is the create/update payload from the dashboard.
type AlertRuleInput struct {
	Name       string  `json:"name"`
	Metric     string  `json:"metric"`
	Comparator string  `json:"comparator"` // gt | lt
	Threshold  float64 `json:"threshold"`
	AgentID    *string `json:"agentId"` // nil/empty = global (all agents)
	Enabled    bool    `json:"enabled"`
}

// AlertRule is the JSON-friendly rule returned to the dashboard.
type AlertRule struct {
	ID         int64     `json:"id"`
	Name       string    `json:"name"`
	Metric     string    `json:"metric"`
	Comparator string    `json:"comparator"`
	Threshold  float64   `json:"threshold"`
	AgentID    *string   `json:"agentId"`
	Enabled    bool      `json:"enabled"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

// AlertEvent is a firing/resolved transition returned to the dashboard.
type AlertEvent struct {
	ID            int64     `json:"id"`
	RuleID        int64     `json:"ruleId"`
	RuleName      string    `json:"ruleName"`
	Metric        string    `json:"metric"`
	AgentID       string    `json:"agentId"`
	State         string    `json:"state"`
	Value         float64   `json:"value"`
	CreatedAt     time.Time `json:"createdAt"`
	WebhookStatus string    `json:"webhookStatus"`
	WebhookError  string    `json:"webhookError,omitempty"`
}

type AlertsService struct {
	queries *db.Queries
}

func NewAlertsService(q *db.Queries) *AlertsService {
	return &AlertsService{queries: q}
}

func (in AlertRuleInput) validate() error {
	if strings.TrimSpace(in.Name) == "" {
		return fmt.Errorf("%w: name is required", ErrInvalidRule)
	}
	if !AlertMetrics[in.Metric] {
		return fmt.Errorf("%w: unknown metric %q", ErrInvalidRule, in.Metric)
	}
	if in.Comparator != "gt" && in.Comparator != "lt" {
		return fmt.Errorf("%w: comparator must be gt or lt", ErrInvalidRule)
	}
	if math.IsNaN(in.Threshold) || math.IsInf(in.Threshold, 0) || in.Threshold < 0 {
		return fmt.Errorf("%w: threshold must be a finite number >= 0", ErrInvalidRule)
	}
	return nil
}

func (s *AlertsService) CreateRule(ctx context.Context, in AlertRuleInput) (AlertRule, error) {
	if s.queries == nil {
		return AlertRule{}, ErrNoDatabase
	}
	if err := in.validate(); err != nil {
		return AlertRule{}, err
	}
	row, err := s.queries.CreateAlertRule(ctx, db.CreateAlertRuleParams{
		Name:       in.Name,
		Metric:     in.Metric,
		Comparator: in.Comparator,
		Threshold:  in.Threshold,
		AgentID:    agentText(in.AgentID),
		Enabled:    in.Enabled,
	})
	if err != nil {
		return AlertRule{}, err
	}
	return mapRule(row), nil
}

func (s *AlertsService) ListRules(ctx context.Context) ([]AlertRule, error) {
	if s.queries == nil {
		return nil, ErrNoDatabase
	}
	rows, err := s.queries.ListAlertRules(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]AlertRule, 0, len(rows))
	for _, r := range rows {
		out = append(out, mapRule(r))
	}
	return out, nil
}

func (s *AlertsService) UpdateRule(ctx context.Context, id int64, in AlertRuleInput) (AlertRule, error) {
	if s.queries == nil {
		return AlertRule{}, ErrNoDatabase
	}
	if err := in.validate(); err != nil {
		return AlertRule{}, err
	}
	row, err := s.queries.UpdateAlertRule(ctx, db.UpdateAlertRuleParams{
		ID:         id,
		Name:       in.Name,
		Metric:     in.Metric,
		Comparator: in.Comparator,
		Threshold:  in.Threshold,
		AgentID:    agentText(in.AgentID),
		Enabled:    in.Enabled,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return AlertRule{}, ErrRuleNotFound
	}
	if err != nil {
		return AlertRule{}, err
	}
	return mapRule(row), nil
}

func (s *AlertsService) DeleteRule(ctx context.Context, id int64) error {
	if s.queries == nil {
		return ErrNoDatabase
	}
	n, err := s.queries.DeleteAlertRule(ctx, id)
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrRuleNotFound
	}
	return nil
}

func (s *AlertsService) ListActive(ctx context.Context) ([]AlertEvent, error) {
	if s.queries == nil {
		return nil, ErrNoDatabase
	}
	rows, err := s.queries.ListActiveAlerts(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]AlertEvent, 0, len(rows))
	for _, r := range rows {
		out = append(out, AlertEvent{
			ID: r.ID, RuleID: r.RuleId, RuleName: r.RuleName, Metric: r.Metric,
			AgentID: r.AgentId, State: r.State, Value: r.Value,
			CreatedAt: r.CreatedAt.Time, WebhookStatus: r.WebhookStatus,
			WebhookError: textOrEmpty(r.WebhookError),
		})
	}
	return out, nil
}

func (s *AlertsService) ListHistory(ctx context.Context, limit int32) ([]AlertEvent, error) {
	if s.queries == nil {
		return nil, ErrNoDatabase
	}
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.queries.ListAlertEvents(ctx, limit)
	if err != nil {
		return nil, err
	}
	out := make([]AlertEvent, 0, len(rows))
	for _, r := range rows {
		out = append(out, AlertEvent{
			ID: r.ID, RuleID: r.RuleId, RuleName: r.RuleName, Metric: r.Metric,
			AgentID: r.AgentId, State: r.State, Value: r.Value,
			CreatedAt: r.CreatedAt.Time, WebhookStatus: r.WebhookStatus,
			WebhookError: textOrEmpty(r.WebhookError),
		})
	}
	return out, nil
}

func mapRule(r db.AlertRule) AlertRule {
	return AlertRule{
		ID:         r.ID,
		Name:       r.Name,
		Metric:     r.Metric,
		Comparator: r.Comparator,
		Threshold:  r.Threshold,
		AgentID:    toAgentPtr(r.AgentID),
		Enabled:    r.Enabled,
		CreatedAt:  r.CreatedAt.Time,
		UpdatedAt:  r.UpdatedAt.Time,
	}
}

func toAgentPtr(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	s := t.String
	return &s
}

// agentText treats nil or empty agentId as a global rule (SQL NULL).
func agentText(p *string) pgtype.Text {
	if p == nil || *p == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *p, Valid: true}
}

func textOrEmpty(t pgtype.Text) string {
	if !t.Valid {
		return ""
	}
	return t.String
}
