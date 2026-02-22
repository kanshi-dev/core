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
