package service

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/kanshi-dev/core/internal/db"
)

type MetricsService struct {
	queries *db.Queries
}

func NewMetricsService(q *db.Queries) *MetricsService {
	return &MetricsService{queries: q}
}

func (s *MetricsService) GetMetrics(
	ctx context.Context,
	agentID string,
	name string,
	from time.Time,
	to time.Time,
) ([]db.GetMetricsByTimeRangeRow, error) {

	if s.queries == nil {
		return nil, ErrNoDatabase
	}

	return s.queries.GetMetricsByTimeRange(
		ctx,
		db.GetMetricsByTimeRangeParams{
			AgentID: agentID,
			Name:    name,
			FromTs:  pgtype.Timestamptz{Time: from, Valid: true},
			ToTs:    pgtype.Timestamptz{Time: to, Valid: true},
		},
	)
}

func (s *MetricsService) GetAggregatedMetrics(
	ctx context.Context,
	agentID string,
	name string,
	interval pgtype.Interval,
	from time.Time,
	to time.Time,
) ([]db.GetAggregatedMetricsRow, error) {

	if s.queries == nil {
		return nil, ErrNoDatabase
	}

	return s.queries.GetAggregatedMetrics(
		ctx,
		db.GetAggregatedMetricsParams{
			AgentID:  agentID,
			Name:     name,
			Interval: interval,
			FromTs:   pgtype.Timestamptz{Time: from, Valid: true},
			ToTs:     pgtype.Timestamptz{Time: to, Valid: true},
		},
	)
}
