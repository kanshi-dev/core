package ingest

import (
	"math"
	"testing"
	"time"

	pb "github.com/kanshi-dev/core/proto"
)

func TestMetricBoundaryValidation(t *testing.T) {
	now := time.Now()
	valid := &pb.Batch{AgentId: "agent", Points: []*pb.Point{{
		Name: "cpu.used_percent", Value: 1, TimestampUnixNano: now.UnixNano(), Tags: []string{"env:test"},
	}}}
	if err := validateBatch(valid, now); err != nil {
		t.Fatalf("valid batch failed: %v", err)
	}

	valid.Points[0].Value = math.Inf(1)
	if err := validateBatch(valid, now); err == nil {
		t.Fatal("expected non-finite metric value to fail")
	}

	valid.Points[0].Value = 1
	valid.Points = make([]*pb.Point, maxMetricBatch+1)
	if err := validateBatch(valid, now); err == nil {
		t.Fatal("expected oversized metric batch to fail")
	}
}
