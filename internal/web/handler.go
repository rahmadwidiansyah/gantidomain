package web

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"gantibdomain"
	"gantibdomain/internal/config"
)

const (
	contentSecurityPolicy = "default-src 'none'; style-src 'self' 'nonce-%s'; img-src 'self'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'"
	fallbackDescription   = "Kami telah memindahkan website ke alamat baru. Silakan gunakan tautan resmi di halaman ini."
)

type pageData struct {
	Nonce       string
	Status      string
	Title       string
	Description string
	Note        string
	ThemeColor  string
	CSSVersion  string
	IconVersion string
	Project     *config.Project
	Projects    []config.Project
}

// Handler melayani seluruh halaman dan aset statis.
type Handler struct {
	logger      *slog.Logger
	templates   *templates
	static      http.Handler
	projects    []config.Project
	bySlug      map[string]config.Project
	cssVersion  string
	iconVersion string
}

// New menyiapkan handler dari konfigurasi yang sudah tervalidasi.
func New(cfg *config.Config, logger *slog.Logger) (*Handler, error) {
	if logger == nil {
		logger = slog.Default()
	}

	tpl, err := parseTemplates()
	if err != nil {
		return nil, err
	}

	staticRoot, err := fs.Sub(assets.Static, "static")
	if err != nil {
		return nil, fmt.Errorf("gagal membuka aset statis: %w", err)
	}

	cssVersion, err := assetVersion(assets.Static, "static/css/style.css")
	if err != nil {
		return nil, err
	}
	iconVersion, err := assetVersion(assets.Static, "static/favicon.svg")
	if err != nil {
		return nil, err
	}

	bySlug := make(map[string]config.Project, len(cfg.Projects))
	for _, p := range cfg.Projects {
		bySlug[p.Slug] = p
	}

	return &Handler{
		logger:      logger,
		templates:   tpl,
		static:      http.StripPrefix("/static/", http.FileServer(http.FS(staticRoot))),
		projects:    cfg.Projects,
		bySlug:      bySlug,
		cssVersion:  cssVersion,
		iconVersion: iconVersion,
	}, nil
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

	h.serve(rec, r)

	h.logger.Info("request",
		"method", r.Method,
		"path", r.URL.Path,
		"status", rec.status,
		"duration_ms", time.Since(start).Milliseconds(),
	)
}

func (h *Handler) serve(w http.ResponseWriter, r *http.Request) {
	nonce, err := newNonce()
	if err != nil {
		h.logger.Error("gagal membuat nonce", "error", err)
		http.Error(w, "Terjadi kesalahan internal.", http.StatusInternalServerError)
		return
	}
	setSecurityHeaders(w, nonce)

	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		h.render(w, r, http.StatusMethodNotAllowed, h.templates.notFound, h.errorPageData(nonce,
			"405",
			"Metode tidak didukung",
			"Halaman ini hanya dapat diakses dengan metode GET atau HEAD.",
			"",
		))
		return
	}

	path := r.URL.Path

	switch path {
	case "/":
		h.render(w, r, http.StatusOK, h.templates.index, h.indexPageData(nonce))
		return
	case "/healthz":
		writeHealth(w, r)
		return
	}

	if strings.HasPrefix(path, "/static/") {
		// Jangan pernah menampilkan directory listing.
		if strings.HasSuffix(path, "/") {
			h.renderNotFound(w, r, nonce)
			return
		}
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		h.static.ServeHTTP(w, r)
		return
	}

	rest := strings.TrimPrefix(path, "/")

	// Normalisasi trailing slash: /project-a/ -> /project-a (301).
	if strings.HasSuffix(rest, "/") {
		slug := strings.TrimSuffix(rest, "/")
		if slug != "" && !strings.Contains(slug, "/") {
			if _, ok := h.bySlug[slug]; ok {
				target := "/" + slug
				if r.URL.RawQuery != "" {
					target += "?" + r.URL.RawQuery
				}
				w.Header().Set("Cache-Control", "no-store")
				w.Header().Set("Location", target)
				w.WriteHeader(http.StatusMovedPermanently)
				return
			}
		}
		h.renderNotFound(w, r, nonce)
		return
	}

	if strings.Contains(rest, "/") {
		h.renderNotFound(w, r, nonce)
		return
	}

	p, ok := h.bySlug[rest]
	if !ok {
		h.renderNotFound(w, r, nonce)
		return
	}

	h.render(w, r, http.StatusOK, h.templates.migration, h.migrationPageData(nonce, p))
}

func (h *Handler) indexPageData(nonce string) pageData {
	return pageData{
		Nonce:       nonce,
		Title:       "Daftar Project — Pemberitahuan Perpindahan Domain",
		Description: "Daftar project yang alamat websitenya telah berpindah.",
		ThemeColor:  config.DefaultAccentColor,
		CSSVersion:  h.cssVersion,
		IconVersion: h.iconVersion,
		Projects:    h.projects,
	}
}

func (h *Handler) migrationPageData(nonce string, p config.Project) pageData {
	description := p.Description
	if description == "" {
		description = fallbackDescription
	}
	project := p
	return pageData{
		Nonce:       nonce,
		Title:       fmt.Sprintf("%s — Alamat Baru", p.Name),
		Description: description,
		ThemeColor:  p.AccentColor,
		CSSVersion:  h.cssVersion,
		IconVersion: h.iconVersion,
		Project:     &project,
	}
}

func (h *Handler) errorPageData(nonce, status, title, description, note string) pageData {
	return pageData{
		Nonce:       nonce,
		Status:      status,
		Title:       title,
		Description: description,
		Note:        note,
		ThemeColor:  config.DefaultAccentColor,
		CSSVersion:  h.cssVersion,
		IconVersion: h.iconVersion,
	}
}

func (h *Handler) renderNotFound(w http.ResponseWriter, r *http.Request, nonce string) {
	h.render(w, r, http.StatusNotFound, h.templates.notFound, h.errorPageData(nonce,
		"404",
		"Halaman tidak ditemukan",
		"Alamat yang Anda buka tidak tersedia atau sudah tidak digunakan.",
		"Periksa kembali alamat yang Anda tulis, atau kembali ke halaman utama untuk melihat daftar project.",
	))
}

func (h *Handler) render(w http.ResponseWriter, r *http.Request, status int, tpl *template.Template, data pageData) {
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		h.logger.Error("gagal merender template", "error", err)
		http.Error(w, "Terjadi kesalahan internal.", http.StatusInternalServerError)
		return
	}

	header := w.Header()
	header.Set("Content-Type", "text/html; charset=utf-8")
	header.Set("Content-Length", strconv.Itoa(buf.Len()))
	if status == http.StatusOK {
		header.Set("Cache-Control", "no-cache")
	} else {
		header.Set("Cache-Control", "no-store")
	}

	w.WriteHeader(status)
	if r.Method == http.MethodHead {
		return
	}
	_, _ = w.Write(buf.Bytes())
}

func writeHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	if r.Method == http.MethodHead {
		return
	}
	_, _ = io.WriteString(w, "ok\n")
}

func setSecurityHeaders(w http.ResponseWriter, nonce string) {
	header := w.Header()
	header.Set("X-Content-Type-Options", "nosniff")
	header.Set("Referrer-Policy", "strict-origin-when-cross-origin")
	header.Set("X-Frame-Options", "DENY")
	header.Set("Content-Security-Policy", fmt.Sprintf(contentSecurityPolicy, nonce))
	header.Set("Content-Language", "id")
	header.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
}

func newNonce() (string, error) {
	buf := make([]byte, 18)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

type statusRecorder struct {
	http.ResponseWriter
	status int
	wrote  bool
}

func (r *statusRecorder) WriteHeader(code int) {
	if r.wrote {
		return
	}
	r.wrote = true
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	if !r.wrote {
		r.WriteHeader(http.StatusOK)
	}
	return r.ResponseWriter.Write(b)
}
