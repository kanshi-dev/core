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

func TestProfileBoundaryValidation(t *testing.T) {
	report := &pb.AgentReport{AgentId: "agent", Hostname: "host", ProfileTargets: []*pb.ProfileTarget{
		{Name: "checkout"}, {Name: "discovered-6060", Discovered: true},
	}}
	if err := validateReport(report); err != nil {
		t.Fatalf("valid targets failed: %v", err)
	}
	report.ProfileTargets[1].Name = "checkout"
	if err := validateReport(report); err == nil {
		t.Fatal("expected duplicate target to fail")
	}
	report.ProfileTargets[1].Name = "bad target"
	if err := validateReport(report); err == nil {
		t.Fatal("expected invalid target name to fail")
	}

	upload := &pb.ProfileUpload{CaptureId: "capture", AgentId: "agent", Artifact: []byte("profile"), Filename: "cpu.pb.gz", ContentType: "application/octet-stream"}
	if err := validateProfileUpload(upload); err != nil {
		t.Fatalf("valid upload failed: %v", err)
	}
	upload.Artifact = make([]byte, maxProfileArtifact+1)
	if err := validateProfileUpload(upload); err == nil {
		t.Fatal("expected oversized profile to fail")
	}
	upload.Artifact = []byte("profile")
	upload.Error = "capture failed"
	if err := validateProfileUpload(upload); err == nil {
		t.Fatal("expected artifact and error together to fail")
	}
}
