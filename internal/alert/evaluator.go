// Package alert evaluates persisted alert rules on a fixed schedule, records
// firing/resolved transitions, and dispatches them to webhooks. The
// alert_events table is the persisted state: the latest event per (rule, agent)
// is the current state, which is read back on every tick and after a restart.
package alert

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/kanshi-dev/core/internal/db"
)

const offlineMetric = "agent.offline"

type transition int

const (
	noop transition = iota
	fire
	resolve
)

// decide returns the transition for a target given whether its latest event was
// firing and whether it is currently breaching. A firing state is held (no
// duplicate events) until the breach clears.
func decide(priorFiring, breaching bool) transition {
	switch {
	case breaching && !priorFiring:
		return fire
	case !breaching && priorFiring:
		return resolve
	default:
		return noop
	}
}

// breach reports whether value breaches threshold under the comparator.
func breach(comparator string, value, threshold float64) bool {
	if comparator == "lt" {
		return value < threshold
	}
	return value > threshold
}

type Evaluator struct {
	q        *db.Queries
	webhooks *Dispatcher
	interval time.Duration
}

func NewEvaluator(q *db.Queries, webhooks *Dispatcher, interval time.Duration) *Evaluator {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	return &Evaluator{q: q, webhooks: webhooks, interval: interval}
}

func (e *Evaluator) Run(ctx context.Context) {
	ticker := time.NewTicker(e.interval)
	defer ticker.Stop()
	log.Printf("alert evaluator running every %s", e.interval)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := e.evaluate(ctx); err != nil {
				log.Printf("alert evaluation failed: %v", err)
			}
		}
	}
}

func (e *Evaluator) evaluate(ctx context.Context) error {
	rules, err := e.q.ListEnabledAlertRules(ctx)
	if err != nil {
		return err
	}
	if len(rules) == 0 {
		return nil
	}

	prior, err := e.priorStates(ctx)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	metricCache := map[string]map[string]float64{}
	var agents []db.ListAgentsRow
	agentsLoaded := false

	for _, r := range rules {
		if r.Metric == offlineMetric {
			if !agentsLoaded {
				if agents, err = e.q.ListAgents(ctx); err != nil {
					return err
				}
				agentsLoaded = true
			}
			for _, a := range targetsOffline(r, agents) {
				secs := now.Sub(a.LastSeen.Time).Seconds()
				e.step(ctx, r, a.AgentId, secs, secs > r.Threshold, prior)
			}
			continue
		}

		latest, ok := metricCache[r.Metric]
		if !ok {
			rows, err := e.q.GetLatestMetricPerAgent(ctx, r.Metric)
			if err != nil {
				return err
			}
			latest = make(map[string]float64, len(rows))
			for _, m := range rows {
				latest[m.AgentId] = m.Value
			}
			metricCache[r.Metric] = latest
		}
		for agentID, value := range targetsMetric(r, latest) {
			e.step(ctx, r, agentID, value, breach(r.Comparator, value, r.Threshold), prior)
		}
	}
	return nil
}

// step applies the transition for one (rule, agent) target and persists it.
func (e *Evaluator) step(ctx context.Context, r db.AlertRule, agentID string, value float64, breaching bool, prior map[string]bool) {
	key := stateKey(r.ID, agentID)
	switch decide(prior[key], breaching) {
	case fire:
		e.transition(ctx, r, agentID, "firing", value)
		prior[key] = true
	case resolve:
		e.transition(ctx, r, agentID, "resolved", value)
		prior[key] = false
	}
}

func (e *Evaluator) transition(ctx context.Context, r db.AlertRule, agentID, state string, value float64) {
	status := "pending"
	if !e.webhooks.enabled() {
		status = "skipped"
	}
	ev, err := e.q.InsertAlertEvent(ctx, db.InsertAlertEventParams{
		RuleID:        r.ID,
		AgentID:       agentID,
		State:         state,
		Value:         value,
		WebhookStatus: status,
	})
	if err != nil {
		log.Printf("alert: insert %s event failed (rule %d agent %s): %v", state, r.ID, agentID, err)
		return
	}
	if e.webhooks.enabled() {
		// Detached context so a bounded retry survives past this tick without
		// blocking evaluation or ingest.
		go e.webhooks.Deliver(context.Background(), r, ev)
	}
}

func (e *Evaluator) priorStates(ctx context.Context) (map[string]bool, error) {
	rows, err := e.q.GetLatestEventPerTarget(ctx)
	if err != nil {
		return nil, err
	}
	m := make(map[string]bool, len(rows))
	for _, row := range rows {
		m[stateKey(row.RuleID, row.AgentID)] = row.State == "firing"
	}
	return m, nil
}

// targetsMetric resolves a metric rule to the agents it applies to. A global
// rule (no agent_id) covers every agent with a fresh metric value.
func targetsMetric(r db.AlertRule, latest map[string]float64) map[string]float64 {
	if r.AgentID.Valid {
		if v, ok := latest[r.AgentID.String]; ok {
			return map[string]float64{r.AgentID.String: v}
		}
		return nil
	}
	return latest
}

func targetsOffline(r db.AlertRule, agents []db.ListAgentsRow) []db.ListAgentsRow {
	if !r.AgentID.Valid {
		return agents
	}
	for _, a := range agents {
		if a.AgentId == r.AgentID.String {
			return []db.ListAgentsRow{a}
		}
	}
	return nil
}

func stateKey(ruleID int64, agentID string) string {
	return fmt.Sprintf("%d|%s", ruleID, agentID)
}
