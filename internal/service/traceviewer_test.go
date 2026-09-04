package service

import "testing"

func TestRewriteTraceAssetURLs(t *testing.T) {
	got := string(rewriteTraceAssetURLs([]byte(`<a href="/trace?view=proc"><script src="/static/app.js"></script>`), "/api/v1/profiles/id/trace"))
	want := `<a href="/api/v1/profiles/id/trace/trace?view=proc"><script src="/api/v1/profiles/id/trace/static/app.js"></script>`
	if got != want {
		t.Fatalf("rewrite = %q, want %q", got, want)
	}
}
