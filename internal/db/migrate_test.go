package db

import (
	"strings"
	"testing"
)

func TestProfileCaptureBoundsAreMigrated(t *testing.T) {
	for _, want := range []string{
		"state IN ('queued', 'capturing', 'completed', 'failed')",
		"OCTET_LENGTH(artifact) <= 10485760",
		"WHERE state IN ('queued', 'capturing')",
		"DEFAULT NOW() + INTERVAL '5 minutes'",
	} {
		if !strings.Contains(migrations, want) {
			t.Fatalf("migration missing %q", want)
		}
	}
}

func TestProfileCaptureQueriesPreserveLifecycle(t *testing.T) {
	for name, query := range map[string]string{
		"create":   createProfileCapture,
		"get":      getProfileCapture,
		"list":     listProfileCapturesByAgent,
		"start":    startProfileCapture,
		"complete": completeProfileCapture,
		"fail":     failProfileCapture,
	} {
		if !strings.Contains(query, "DELETE FROM profile_captures WHERE expires_at <= NOW()") {
			t.Errorf("%s does not clean up expired captures", name)
		}
	}

	for name, query := range map[string]string{
		"start":    startProfileCapture,
		"complete": completeProfileCapture,
		"fail":     failProfileCapture,
	} {
		if !strings.Contains(query, "profile_captures.agent_id = $") {
			t.Errorf("%s does not enforce Agent ownership", name)
		}
		if !strings.Contains(query, "expires_at > NOW()") {
			t.Errorf("%s permits expired transitions", name)
		}
	}

	for name, query := range map[string]string{"complete": completeProfileCapture, "fail": failProfileCapture} {
		if !strings.Contains(query, "expires_at = NOW() + INTERVAL '24 hours'") {
			t.Errorf("%s does not apply artifact retention", name)
		}
	}
}
