// Package config membaca dan memvalidasi konfigurasi project dari environment
// variable. Konfigurasi dimuat sekali saat startup dan tidak dapat diubah
// melalui HTTP.
package config

import (
	"errors"
	"fmt"
	"math"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
)

const (
	// DefaultPort dipakai bila PORT tidak diatur.
	DefaultPort = "8080"
	// DefaultAccentColor dipakai bila ACCENT_COLOR tidak diatur.
	DefaultAccentColor = "#1d4ed8"

	maxURLLength  = 2048
	maxSlugLength = 64
)

var (
	// Slug hanya boleh huruf kecil, angka, dan tanda hubung, serta tidak boleh
	// diawali atau diakhiri tanda hubung.
	slugPattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,62}[a-z0-9])?$`)
	// Warna hanya boleh hex pendek atau panjang.
	colorPattern = regexp.MustCompile(`^#(?:[0-9a-fA-F]{3}|[0-9a-fA-F]{6})$`)
)

// Project adalah satu konfigurasi landing page.
type Project struct {
	Slug            string
	Name            string
	OldDomain       string
	NewURL          string
	AppURL          string
	Description     string
	Message         string
	AccentColor     string
	AccentTextColor string
}

// Config adalah konfigurasi aplikasi yang sudah tervalidasi.
type Config struct {
	Port     string
	Projects []Project

	bySlug map[string]Project
}

// Project mencari project berdasarkan slug.
func (c *Config) Project(slug string) (Project, bool) {
	p, ok := c.bySlug[slug]
	return p, ok
}

// Load membaca konfigurasi dari environment proses.
func Load() (*Config, error) {
	return load(os.Getenv)
}

func load(getenv func(string) string) (*Config, error) {
	port := strings.TrimSpace(getenv("PORT"))
	if port == "" {
		port = DefaultPort
	}
	if n, err := strconv.Atoi(port); err != nil || n < 1 || n > 65535 {
		return nil, fmt.Errorf("PORT %q tidak valid: harus berupa angka 1-65535", port)
	}

	cfg := &Config{
		Port:     port,
		bySlug:   make(map[string]Project),
		Projects: []Project{},
	}

	var errs []error
	seen := make(map[string]bool)

	for _, entry := range strings.Split(getenv("PROJECTS"), ",") {
		slug := strings.TrimSpace(entry)
		if slug == "" {
			continue
		}
		if seen[slug] {
			errs = append(errs, fmt.Errorf("project %q: slug duplikat pada PROJECTS", slug))
			continue
		}
		seen[slug] = true

		if !slugPattern.MatchString(slug) || len(slug) > maxSlugLength {
			errs = append(errs, fmt.Errorf(
				"project %q: slug tidak valid, gunakan huruf kecil a-z, angka 0-9, dan tanda hubung (maksimal %d karakter, tidak diawali/diakhiri tanda hubung)",
				slug, maxSlugLength,
			))
			continue
		}

		p, err := loadProject(slug, getenv)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		cfg.Projects = append(cfg.Projects, p)
		cfg.bySlug[slug] = p
	}

	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	return cfg, nil
}

func loadProject(slug string, getenv func(string) string) (Project, error) {
	prefix := envPrefix(slug)
	get := func(key string) string {
		return strings.TrimSpace(getenv(prefix + key))
	}

	p := Project{
		Slug:        slug,
		Name:        get("NAME"),
		OldDomain:   get("OLD_DOMAIN"),
		NewURL:      get("NEW_URL"),
		AppURL:      get("APP_URL"),
		Description: get("DESCRIPTION"),
		Message:     get("MESSAGE"),
		AccentColor: get("ACCENT_COLOR"),
	}

	var errs []error

	if p.Name == "" {
		errs = append(errs, fmt.Errorf("%sNAME wajib diisi", prefix))
	}

	if p.NewURL == "" {
		errs = append(errs, fmt.Errorf("%sNEW_URL wajib diisi", prefix))
	} else if normalized, err := validateURL(p.NewURL); err != nil {
		errs = append(errs, fmt.Errorf("%sNEW_URL tidak valid: %w", prefix, err))
	} else {
		p.NewURL = normalized
	}

	if p.AppURL != "" {
		normalized, err := validateURL(p.AppURL)
		if err != nil {
			errs = append(errs, fmt.Errorf("%sAPP_URL tidak valid: %w", prefix, err))
		} else {
			p.AppURL = normalized
		}
	}

	if p.AccentColor == "" {
		p.AccentColor = DefaultAccentColor
	}
	if !colorPattern.MatchString(p.AccentColor) {
		errs = append(errs, fmt.Errorf(
			"%sACCENT_COLOR %q tidak valid: gunakan format #RGB atau #RRGGBB",
			prefix, p.AccentColor,
		))
	} else {
		p.AccentColor = strings.ToLower(p.AccentColor)
		p.AccentTextColor = readableTextColor(p.AccentColor)
	}

	if len(errs) > 0 {
		return Project{}, fmt.Errorf("project %q tidak valid:\n%w", slug, errors.Join(errs...))
	}
	return p, nil
}

// envPrefix memetakan slug "project-a" menjadi "PROJECT_PROJECT_A_".
func envPrefix(slug string) string {
	return "PROJECT_" + strings.ToUpper(strings.ReplaceAll(slug, "-", "_")) + "_"
}

func validateURL(raw string) (string, error) {
	if len(raw) > maxURLLength {
		return "", fmt.Errorf("panjang melebihi %d karakter", maxURLLength)
	}
	if strings.ContainsAny(raw, " \t\r\n") {
		return "", errors.New("mengandung spasi atau karakter baris baru")
	}
	for _, r := range raw {
		if r < 0x20 || r == 0x7f {
			return "", errors.New("mengandung karakter kontrol")
		}
	}

	u, err := url.Parse(raw)
	if err != nil {
		return "", errors.New("format URL tidak dapat dibaca")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("skema %q tidak diizinkan, gunakan http atau https", u.Scheme)
	}
	if u.Host == "" {
		return "", errors.New("host wajib ada")
	}
	if u.User != nil {
		return "", errors.New("URL tidak boleh mengandung kredensial")
	}
	return u.String(), nil
}

// readableTextColor memilih warna teks hitam atau putih yang kontrasnya lebih
// tinggi terhadap warna aksen. Salah satu pilihan selalu memenuhi WCAG AA
// untuk teks normal (>= 4.5:1).
func readableTextColor(hexColor string) string {
	r, g, b := parseHexColor(hexColor)
	l := relativeLuminance(r, g, b)
	if contrastRatio(l, 0) >= contrastRatio(l, 1) {
		return "#0b0b0b"
	}
	return "#ffffff"
}

func parseHexColor(s string) (float64, float64, float64) {
	s = strings.TrimPrefix(s, "#")
	if len(s) == 3 {
		s = string([]byte{s[0], s[0], s[1], s[1], s[2], s[2]})
	}

	var channels [3]float64
	for i := range 3 {
		n, err := strconv.ParseUint(s[i*2:i*2+2], 16, 8)
		if err != nil {
			return 0, 0, 0
		}
		c := float64(n) / 255
		if c <= 0.04045 {
			c /= 12.92
		} else {
			c = math.Pow((c+0.055)/1.055, 2.4)
		}
		channels[i] = c
	}
	return channels[0], channels[1], channels[2]
}

func relativeLuminance(r, g, b float64) float64 {
	return 0.2126*r + 0.7152*g + 0.0722*b
}

func contrastRatio(l1, l2 float64) float64 {
	if l1 < l2 {
		l1, l2 = l2, l1
	}
	return (l1 + 0.05) / (l2 + 0.05)
}
