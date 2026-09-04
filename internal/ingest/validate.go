package ingest

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"time"

	pb "github.com/kanshi-dev/core/proto"
)

const (
	maxMetricBatch     = 1000
	maxMetricTags      = 32
	maxFieldBytes      = 255
	maxProfileTargets  = 20
	maxProfileArtifact = 10 << 20
)

var profileTargetName = regexp.MustCompile(`^[A-Za-z0-9._-]{1,64}$`)

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
	if len(req.GetProfileTargets()) > maxProfileTargets {
		return fmt.Errorf("profile targets exceed %d entries", maxProfileTargets)
	}
	seen := make(map[string]struct{}, len(req.GetProfileTargets()))
	for _, target := range req.GetProfileTargets() {
		if target == nil || !profileTargetName.MatchString(target.GetName()) {
			return errors.New("profile target name must match [A-Za-z0-9._-]{1,64}")
		}
		if _, ok := seen[target.GetName()]; ok {
			return errors.New("profile target names must be unique")
		}
		seen[target.GetName()] = struct{}{}
	}
	return nil
}

func validateProfileUpload(req *pb.ProfileUpload) error {
	if req.GetCaptureId() == "" || len(req.GetCaptureId()) > maxFieldBytes {
		return errors.New("capture_id must be 1 to 255 bytes")
	}
	if req.GetAgentId() == "" || len(req.GetAgentId()) > maxFieldBytes {
		return errors.New("agent_id must be 1 to 255 bytes")
	}
	if len(req.GetArtifact()) > maxProfileArtifact {
		return errors.New("profile artifact exceeds 10 MiB")
	}
	if req.GetError() == "" && (req.GetFilename() == "" || req.GetContentType() == "") {
		return errors.New("profile filename and content_type are required")
	}
	if len(req.GetFilename()) > maxFieldBytes || len(req.GetContentType()) > maxFieldBytes {
		return errors.New("profile metadata exceeds 255 bytes")
	}
	if len(req.GetError()) > 4096 {
		return errors.New("profile error exceeds 4096 bytes")
	}
	if req.GetError() == "" && len(req.GetArtifact()) == 0 {
		return errors.New("profile artifact is required when error is empty")
	}
	if req.GetError() != "" && len(req.GetArtifact()) != 0 {
		return errors.New("profile upload cannot contain both artifact and error")
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
