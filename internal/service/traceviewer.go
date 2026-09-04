package service

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

// TraceViewerService runs the Go trace viewer bundled with Core and proxies it
// behind the authenticated API. Only one viewer is kept alive at a time.
type TraceViewerService struct {
	profiles *ProfilesService
	mu       sync.Mutex
	active   *traceViewer
	sessions map[string]traceSession
}

type traceSession struct {
	id      string
	expires time.Time
}

type traceViewer struct {
	id     string
	cancel context.CancelFunc
	proxy  *httputil.ReverseProxy
	dir    string
}

func NewTraceViewerService(profiles *ProfilesService) *TraceViewerService {
	return &TraceViewerService{profiles: profiles, sessions: make(map[string]traceSession)}
}

func (s *TraceViewerService) CreateSession(ctx context.Context, id string) (string, error) {
	if _, err := s.profiles.Download(ctx, id); err != nil {
		return "", err
	}
	var raw [24]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	token := fmt.Sprintf("%x", raw[:])
	s.mu.Lock()
	s.sessions[token] = traceSession{id: id, expires: time.Now().Add(5 * time.Minute)}
	s.mu.Unlock()
	return token, nil
}

func (s *TraceViewerService) AuthorizedSession(token, id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	session, ok := s.sessions[token]
	if !ok || session.id != id || time.Now().After(session.expires) {
		delete(s.sessions, token)
		return false
	}
	return true
}

func (s *TraceViewerService) Handler(id, prefix string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		viewer, err := s.viewer(r.Context(), id, prefix)
		if err != nil {
			http.Error(w, "trace viewer unavailable", http.StatusInternalServerError)
			return
		}
		path := strings.TrimPrefix(r.URL.Path, prefix)
		if path == "" {
			path = "/"
		}
		if !strings.HasPrefix(path, "/") {
			path = "/" + path
		}
		r.URL.Path = path
		viewer.proxy.ServeHTTP(w, r)
	})
}

func (s *TraceViewerService) viewer(ctx context.Context, id, prefix string) (*traceViewer, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.active != nil && s.active.id == id {
		return s.active, nil
	}
	if s.active != nil {
		s.active.cancel()
		os.RemoveAll(s.active.dir)
		s.active = nil
	}
	artifact, err := s.profiles.Download(ctx, id)
	if err != nil {
		return nil, err
	}
	dir, err := os.MkdirTemp("", "kanshi-trace-")
	if err != nil {
		return nil, err
	}
	path := filepath.Join(dir, "trace.out")
	if err := os.WriteFile(path, artifact.Data, 0o600); err != nil {
		os.RemoveAll(dir)
		return nil, err
	}
	viewerPath := os.Getenv("KANSHI_TRACE_VIEWER_BIN")
	if viewerPath == "" {
		viewerPath = "/app/kanshi-trace-viewer"
	}
	processCtx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(processCtx, viewerPath, "-http=127.0.0.1:0", path)
	stderr, err := cmd.StderrPipe()
	if err != nil {
		cancel()
		os.RemoveAll(dir)
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		cancel()
		os.RemoveAll(dir)
		return nil, err
	}
	port, err := traceViewerPort(stderr)
	if err != nil {
		cancel()
		_ = cmd.Wait()
		os.RemoveAll(dir)
		return nil, err
	}
	target, _ := url.Parse("http://127.0.0.1:" + port)
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.ModifyResponse = func(resp *http.Response) error {
		if strings.HasPrefix(resp.Header.Get("Content-Type"), "text/") || strings.Contains(resp.Header.Get("Content-Type"), "javascript") {
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return err
			}
			_ = resp.Body.Close()
			body = rewriteTraceAssetURLs(body, prefix)
			resp.Body = io.NopCloser(bytes.NewReader(body))
			resp.ContentLength = int64(len(body))
			resp.Header.Set("Content-Length", fmt.Sprint(len(body)))
		}
		return nil
	}
	proxy.ErrorHandler = func(w http.ResponseWriter, _ *http.Request, err error) {
		if !strings.Contains(err.Error(), "context canceled") {
			http.Error(w, "trace viewer unavailable", http.StatusBadGateway)
		}
	}
	go func() { _ = cmd.Wait() }()
	s.active = &traceViewer{id: id, cancel: cancel, proxy: proxy, dir: dir}
	return s.active, nil
}

var traceViewerURL = regexp.MustCompile(`listening on http://127\.0\.0\.1:([0-9]+)`)

func traceViewerPort(r io.Reader) (string, error) {
	scanner := bufio.NewScanner(r)
	lines := make(chan string)
	go func() {
		for scanner.Scan() {
			lines <- scanner.Text()
		}
		close(lines)
	}()
	deadline := time.After(10 * time.Second)
	for {
		select {
		case text, ok := <-lines:
			if !ok {
				return "", fmt.Errorf("trace viewer exited before listening")
			}
			if match := traceViewerURL.FindStringSubmatch(text); len(match) == 2 {
				return match[1], nil
			}
		case <-deadline:
			return "", fmt.Errorf("trace viewer startup timed out")
		}
	}
}

// rewriteTraceAssetURLs keeps the bundled viewer working below the API prefix.
// The viewer emits root-relative links; callers may use this helper when a
// proxy response needs rewriting.
func rewriteTraceAssetURLs(body []byte, prefix string) []byte {
	return bytes.ReplaceAll(body, []byte(`"/`), []byte(`"`+prefix+`/`))
}
