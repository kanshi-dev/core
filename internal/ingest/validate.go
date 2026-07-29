package ingest

import (
	"errors"
	"fmt"
	"math"
	"time"

	pb "github.com/kanshi-dev/core/proto"
)

const (
	maxMetricBatch = 1000
	maxMetricTags  = 32
	maxFieldBytes  = 255
)

func validateReport(req *pb.AgentReport) error {
	if req.GetAgentId() == "" || len(req.GetAgentId()) > maxFieldBytes {
		return errors.New("agent_id must be 1 to 255 bytes")
	}
	if req.GetHostname() == "" || len(req.GetHostname()) > maxFieldBytes {
		return errors.New("hostname must be 1 to 255 bytes")
	}
	if req.GetCpuCores() < 0 || req.GetTotalMemory() < 0 || req.GetDiskSize() < 0 {
		return errors.New("host capacities cannot be negative")
	}
	for name, value := range map[string]string{
		"os": req.GetOs(), "platform": req.GetPlatform(), "arch": req.GetArch(), "version": req.GetVersion(),
	} {
		if len(value) > maxFieldBytes {
			return fmt.Errorf("%s exceeds 255 bytes", name)
		}
	}
	return nil
}

func validateBatch(req *pb.Batch, now time.Time) error {
	if req.GetAgentId() == "" || len(req.GetAgentId()) > maxFieldBytes {
		return errors.New("agent_id must be 1 to 255 bytes")
	}
	if len(req.GetPoints()) > maxMetricBatch {
		return fmt.Errorf("batch exceeds %d points", maxMetricBatch)
	}
	for _, point := range req.GetPoints() {
		if point.GetName() == "" || len(point.GetName()) > maxFieldBytes {
			return errors.New("metric name must be 1 to 255 bytes")
		}
		if math.IsNaN(point.GetValue()) || math.IsInf(point.GetValue(), 0) {
			return errors.New("metric value must be finite")
		}
		if point.GetTimestampUnixNano() <= 0 {
			return errors.New("metric timestamp is required")
		}
		ts := time.Unix(0, point.GetTimestampUnixNano())
		if ts.After(now.Add(5*time.Minute)) || ts.Before(now.Add(-30*24*time.Hour)) {
			return errors.New("metric timestamp is outside the accepted 30-day window")
		}
		if len(point.GetTags()) > maxMetricTags {
			return fmt.Errorf("metric tags exceed %d entries", maxMetricTags)
		}
		for _, tag := range point.GetTags() {
			if len(tag) > maxFieldBytes {
				return errors.New("metric tag cannot exceed 255 bytes")
			}
		}
	}
	return nil
}
