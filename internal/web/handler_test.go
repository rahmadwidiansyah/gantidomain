package web

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"gantibdomain/internal/config"
)

func handlerEnv() map[string]string {
	return map[string]string{
		"PORT":                           "8080",
		"PROJECTS":                       "project-a,project-b",
		"PROJECT_PROJECT_A_NAME":         "Project A",
		"PROJECT_PROJECT_A_OLD_DOMAIN":   "old-a.example",
		"PROJECT_PROJECT_A_NEW_URL":      "https://new-a.example/",
		"PROJECT_PROJECT_A_APP_URL":      "https://downloads.example/app-a",
		"PROJECT_PROJECT_A_DESCRIPTION":  "Website resmi Project A telah berpindah.",
		"PROJECT_PROJECT_A_MESSAGE":      "Gunakan alamat baru.",
		"PROJECT_PROJECT_A_ACCENT_COLOR": "#1d4ed8",
		"PROJECT_PROJECT_B_NAME":         "Project B",
		"PROJECT_PROJECT_B_OLD_DOMAIN":   "old-b.example",
		"PROJECT_PROJECT_B_NEW_URL":      "https://new-b.example",
		"PROJECT_PROJECT_B_APP_URL":      "",
		"PROJECT_PROJECT_B_DESCRIPTION":  "Alamat resmi Project B telah diperbarui.",
	}
}

func newTestHandler(t *testing.T, overrides map[string]string) http.Handler {
	t.Helper()

	env := handlerEnv()
	for key, value := range overrides {
		if value == "" {
			delete(env, key)
			continue
		}
		env[key] = value
	}
	for key, value := range env {
		t.Setenv(key, value)
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load() error = %v", err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
	handler, err := New(cfg, logger)
	if err != nil {
		t.Fatalf("web.New() error = %v", err)
	}
	return handler
}

func doRequest(t *testing.T, handler http.Handler, method, target string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(method, target, nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

func TestRoutingTable(t *testing.T) {
	handler := newTestHandler(t, nil)

	tests := []struct {
		method     string
		path       string
		wantStatus int
		wantLoc    string
	}{
		{http.MethodGet, "/", http.StatusOK, ""},
		{http.MethodGet, "/project-a", http.StatusOK, ""},
		{http.MethodGet, "/project-b?utm=email", http.StatusOK, ""},
		{http.MethodGet, "/project-a/", http.StatusMovedPermanently, "/project-a"},
		{http.MethodGet, "/project-a/?ref=abc", http.StatusMovedPermanently, "/project-a?ref=abc"},
		{http.MethodGet, "/unknown", http.StatusNotFound, ""},
		{http.MethodGet, "/unknown/", http.StatusNotFound, ""},
		{http.MethodGet, "/project-a/extra", http.StatusNotFound, ""},
		{http.MethodGet, "/project-a//", http.StatusNotFound, ""},
		{http.MethodGet, "/healthz", http.StatusOK, ""},
		{http.MethodGet, "/static/", http.StatusNotFound, ""},
		{http.MethodGet, "/static/css/", http.StatusNotFound, ""},
		{http.MethodGet, "/static", http.StatusNotFound, ""},
		{http.MethodGet, "/.env", http.StatusNotFound, ""},
		{http.MethodGet, "/nginx/default.conf", http.StatusNotFound, ""},
		{http.MethodPost, "/project-a", http.StatusMethodNotAllowed, ""},
		{http.MethodPut, "/healthz", http.StatusMethodNotAllowed, ""},
		{http.MethodDelete, "/", http.StatusMethodNotAllowed, ""},
		{http.MethodOptions, "/project-a", http.StatusMethodNotAllowed, ""},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			recorder := doRequest(t, handler, tt.method, tt.path)
			if recorder.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", recorder.Code, tt.wantStatus)
			}
			if tt.wantLoc != "" {
				if got := recorder.Header().Get("Location"); got != tt.wantLoc {
					t.Errorf("Location = %q, want %q", got, tt.wantLoc)
				}
			}
		})
	}
}

func TestMigrationPageContent(t *testing.T) {
	handler := newTestHandler(t, nil)

	body := doRequest(t, handler, http.MethodGet, "/project-a").Body.String()
	for _, want := range []string{
		"Project A",
		"https://new-a.example/",
		"https://downloads.example/app-a",
		"Buka Website Baru",
		"Download Aplikasi Baru",
		"old-a.example",
		"Website resmi Project A telah berpindah.",
		"Gunakan alamat baru.",
		"--accent: #1d4ed8",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("body /project-a tidak mengandung %q", want)
		}
	}

	bodyB := doRequest(t, handler, http.MethodGet, "/project-b").Body.String()
	for _, want := range []string{"Project B", "https://new-b.example", "old-b.example"} {
		if !strings.Contains(bodyB, want) {
			t.Errorf("body /project-b tidak mengandung %q", want)
		}
	}
	if strings.Contains(bodyB, "Download Aplikasi Baru") {
		t.Error("tombol download tidak boleh tampil ketika APP_URL kosong")
	}
	if strings.Contains(bodyB, "new-a.example") {
		t.Error("data project-a bocor ke halaman project-b")
	}
	if strings.Contains(body, "new-b.example") {
		t.Error("data project-b bocor ke halaman project-a")
	}
}

func TestIndexListsProjects(t *testing.T) {
	handler := newTestHandler(t, nil)
	body := doRequest(t, handler, http.MethodGet, "/").Body.String()

	for _, want := range []string{`href="/project-a"`, `href="/project-b"`, "Project A", "Project B"} {
		if !strings.Contains(body, want) {
			t.Errorf("body / tidak mengandung %q", want)
		}
	}
}

func TestEscapesConfigurationValues(t *testing.T) {
	handler := newTestHandler(t, map[string]string{
		"PROJECT_PROJECT_A_NAME":        `<script>alert("x")</script> Project`,
		"PROJECT_PROJECT_A_OLD_DOMAIN":  `old.example</dd><script>alert(2)</script>`,
		"PROJECT_PROJECT_A_NEW_URL":     "https://new-a.example/?a=1&b=2",
		"PROJECT_PROJECT_A_DESCRIPTION": `Deskripsi "kutip" & <b>tebal</b>`,
	})

	body := doRequest(t, handler, http.MethodGet, "/project-a").Body.String()

	for _, forbidden := range []string{"<script>alert", "</dd><script>"} {
		if strings.Contains(body, forbidden) {
			t.Errorf("body mengandung HTML mentah %q", forbidden)
		}
	}
	for _, want := range []string{"&lt;script&gt;", "&lt;b&gt;tebal&lt;/b&gt;", "&amp;"} {
		if !strings.Contains(body, want) {
			t.Errorf("body tidak mengandung escaped value %q", want)
		}
	}
}

func TestSecurityHeaders(t *testing.T) {
	handler := newTestHandler(t, nil)

	for _, path := range []string{"/", "/project-a", "/healthz", "/static/css/style.css"} {
		t.Run(path, func(t *testing.T) {
			recorder := doRequest(t, handler, http.MethodGet, path)
			header := recorder.Header()

			if got := header.Get("X-Content-Type-Options"); got != "nosniff" {
				t.Errorf("X-Content-Type-Options = %q", got)
			}
			if got := header.Get("Referrer-Policy"); got != "strict-origin-when-cross-origin" {
				t.Errorf("Referrer-Policy = %q", got)
			}
			if got := header.Get("X-Frame-Options"); got != "DENY" {
				t.Errorf("X-Frame-Options = %q", got)
			}
			if got := header.Get("Content-Language"); got != "id" {
				t.Errorf("Content-Language = %q", got)
			}

			csp := header.Get("Content-Security-Policy")
			for _, want := range []string{"default-src 'none'", "style-src 'self' 'nonce-", "frame-ancestors 'none'"} {
				if !strings.Contains(csp, want) {
					t.Errorf("CSP = %q, tidak mengandung %q", csp, want)
				}
			}
		})
	}
}

func TestCSPNonceMatchesInlineStyle(t *testing.T) {
	handler := newTestHandler(t, nil)
	recorder := doRequest(t, handler, http.MethodGet, "/project-a")

	csp := recorder.Header().Get("Content-Security-Policy")
	match := regexp.MustCompile(`nonce-([A-Za-z0-9_-]+)`).FindStringSubmatch(csp)
	if match == nil {
		t.Fatalf("CSP tidak mengandung nonce: %q", csp)
	}

	if want := `<style nonce="` + match[1] + `">`; !strings.Contains(recorder.Body.String(), want) {
		t.Errorf("body tidak mengandung %q", want)
	}
}

func TestOnlyOneButtonWithoutAppURL(t *testing.T) {
	handler := newTestHandler(t, nil)
	body := doRequest(t, handler, http.MethodGet, "/project-b").Body.String()

	if count := strings.Count(body, "button-primary"); count != 1 {
		t.Errorf("jumlah tombol utama = %d, want 1", count)
	}
	if count := strings.Count(body, `class="button `); count != 1 {
		t.Errorf("jumlah total tombol = %d, want 1", count)
	}
}

func TestHeadRequests(t *testing.T) {
	handler := newTestHandler(t, nil)

	get := doRequest(t, handler, http.MethodGet, "/project-a")
	head := doRequest(t, handler, http.MethodHead, "/project-a")

	if head.Code != http.StatusOK {
		t.Errorf("status HEAD = %d, want 200", head.Code)
	}
	if head.Body.Len() != 0 {
		t.Errorf("body HEAD = %d byte, want 0", head.Body.Len())
	}
	if got, want := head.Header().Get("Content-Length"), get.Header().Get("Content-Length"); got != want || got == "" {
		t.Errorf("Content-Length HEAD = %q, GET = %q", got, want)
	}
	if length, err := strconv.Atoi(head.Header().Get("Content-Length")); err != nil || length <= 0 {
		t.Errorf("Content-Length HEAD tidak valid: %q", head.Header().Get("Content-Length"))
	}

	health := doRequest(t, handler, http.MethodHead, "/healthz")
	if health.Code != http.StatusOK || health.Body.Len() != 0 {
		t.Errorf("HEAD /healthz = %d, body %d byte", health.Code, health.Body.Len())
	}

	redirect := doRequest(t, handler, http.MethodHead, "/project-a/")
	if redirect.Code != http.StatusMovedPermanently || redirect.Header().Get("Location") != "/project-a" {
		t.Errorf("HEAD /project-a/ = %d, Location %q", redirect.Code, redirect.Header().Get("Location"))
	}
}

func TestStaticAssets(t *testing.T) {
	handler := newTestHandler(t, nil)

	tests := []struct {
		path        string
		contentType string
	}{
		{"/static/css/style.css", "text/css"},
		{"/static/favicon.svg", "image/svg+xml"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			recorder := doRequest(t, handler, http.MethodGet, tt.path)
			if recorder.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", recorder.Code)
			}
			if got := recorder.Header().Get("Content-Type"); !strings.HasPrefix(got, tt.contentType) {
				t.Errorf("Content-Type = %q, want prefix %q", got, tt.contentType)
			}
			if got := recorder.Header().Get("Cache-Control"); !strings.Contains(got, "immutable") {
				t.Errorf("Cache-Control = %q, want immutable", got)
			}
		})
	}

	missing := doRequest(t, handler, http.MethodGet, "/static/css/tidak-ada.css")
	if missing.Code != http.StatusNotFound {
		t.Errorf("aset tidak ada: status = %d, want 404", missing.Code)
	}
}

func TestCacheControl(t *testing.T) {
	handler := newTestHandler(t, nil)

	if got := doRequest(t, handler, http.MethodGet, "/project-a").Header().Get("Cache-Control"); got != "no-cache" {
		t.Errorf("Cache-Control halaman = %q, want no-cache", got)
	}
	if got := doRequest(t, handler, http.MethodGet, "/unknown").Header().Get("Cache-Control"); got != "no-store" {
		t.Errorf("Cache-Control 404 = %q, want no-store", got)
	}
	if got := doRequest(t, handler, http.MethodGet, "/project-a/").Header().Get("Cache-Control"); got != "no-store" {
		t.Errorf("Cache-Control redirect = %q, want no-store", got)
	}
}

func TestMethodNotAllowedHeader(t *testing.T) {
	handler := newTestHandler(t, nil)
	recorder := doRequest(t, handler, http.MethodPost, "/project-a")

	if got := recorder.Header().Get("Allow"); got != "GET, HEAD" {
		t.Errorf("Allow = %q, want %q", got, "GET, HEAD")
	}
}

func TestHealthBody(t *testing.T) {
	handler := newTestHandler(t, nil)
	recorder := doRequest(t, handler, http.MethodGet, "/healthz")

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if body := recorder.Body.String(); body != "ok\n" {
		t.Errorf("body = %q, want %q", body, "ok\n")
	}
}
