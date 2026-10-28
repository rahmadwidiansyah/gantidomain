package config

import (
	"strings"
	"testing"
)

func loadWith(t *testing.T, env map[string]string) (*Config, error) {
	t.Helper()
	return load(func(key string) string { return env[key] })
}

func validEnv() map[string]string {
	return map[string]string{
		"PROJECTS":                       "project-a,project-b",
		"PROJECT_PROJECT_A_NAME":         "Project A",
		"PROJECT_PROJECT_A_OLD_DOMAIN":   "old-a.example",
		"PROJECT_PROJECT_A_NEW_URL":      "https://new-a.example/path",
		"PROJECT_PROJECT_A_APP_URL":      "https://downloads.example/app-a",
		"PROJECT_PROJECT_A_DESCRIPTION":  "Deskripsi A",
		"PROJECT_PROJECT_A_MESSAGE":      "Pesan A",
		"PROJECT_PROJECT_A_ACCENT_COLOR": "#1d4ed8",
		"PROJECT_PROJECT_B_NAME":         "Project B",
		"PROJECT_PROJECT_B_NEW_URL":      "https://new-b.example",
	}
}

func TestLoadValid(t *testing.T) {
	cfg, err := loadWith(t, validEnv())
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Port != DefaultPort {
		t.Errorf("Port = %q, want %q", cfg.Port, DefaultPort)
	}
	if len(cfg.Projects) != 2 {
		t.Fatalf("len(Projects) = %d, want 2", len(cfg.Projects))
	}

	a := cfg.Projects[0]
	if a.Slug != "project-a" || a.Name != "Project A" {
		t.Errorf("project pertama = %+v", a)
	}
	if a.NewURL != "https://new-a.example/path" {
		t.Errorf("NewURL = %q", a.NewURL)
	}
	if a.AppURL != "https://downloads.example/app-a" {
		t.Errorf("AppURL = %q", a.AppURL)
	}
	if a.AccentColor != "#1d4ed8" || a.AccentTextColor != "#ffffff" {
		t.Errorf("warna aksen = %q / %q", a.AccentColor, a.AccentTextColor)
	}

	b := cfg.Projects[1]
	if b.AppURL != "" {
		t.Errorf("AppURL project-b = %q, want kosong", b.AppURL)
	}
	if b.AccentColor != DefaultAccentColor {
		t.Errorf("AccentColor default = %q, want %q", b.AccentColor, DefaultAccentColor)
	}

	found, ok := cfg.Project("project-b")
	if !ok || found.Name != "Project B" {
		t.Errorf("Project(project-b) = %+v, %v", found, ok)
	}
	if _, ok := cfg.Project("project-c"); ok {
		t.Error("Project(project-c) seharusnya tidak ditemukan")
	}
}

func TestLoadTrimsProjectList(t *testing.T) {
	env := validEnv()
	env["PROJECTS"] = " project-a , project-b "
	if _, err := loadWith(t, env); err != nil {
		t.Fatalf("Load() error = %v", err)
	}
}

func TestLoadEmptyProjects(t *testing.T) {
	cfg, err := loadWith(t, map[string]string{"PROJECTS": ""})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(cfg.Projects) != 0 {
		t.Errorf("len(Projects) = %d, want 0", len(cfg.Projects))
	}
}

func TestLoadPort(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		env := validEnv()
		env["PORT"] = "9090"
		cfg, err := loadWith(t, env)
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if cfg.Port != "9090" {
			t.Errorf("Port = %q, want 9090", cfg.Port)
		}
	})

	for _, port := range []string{"abc", "0", "-1", "65536", "80.5"} {
		t.Run("invalid_"+port, func(t *testing.T) {
			env := validEnv()
			env["PORT"] = port
			if _, err := loadWith(t, env); err == nil {
				t.Errorf("Load() dengan PORT %q seharusnya gagal", port)
			}
		})
	}
}

func TestLoadInvalidProjects(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(map[string]string)
		wantErr string
	}{
		{
			name:    "name kosong",
			mutate:  func(env map[string]string) { delete(env, "PROJECT_PROJECT_A_NAME") },
			wantErr: "NAME wajib diisi",
		},
		{
			name:    "new url kosong",
			mutate:  func(env map[string]string) { env["PROJECT_PROJECT_A_NEW_URL"] = "" },
			wantErr: "NEW_URL wajib diisi",
		},
		{
			name:    "new url javascript",
			mutate:  func(env map[string]string) { env["PROJECT_PROJECT_A_NEW_URL"] = "javascript:alert(1)" },
			wantErr: "NEW_URL tidak valid",
		},
		{
			name:    "new url tanpa skema",
			mutate:  func(env map[string]string) { env["PROJECT_PROJECT_A_NEW_URL"] = "new-a.example" },
			wantErr: "NEW_URL tidak valid",
		},
		{
			name:    "new url tanpa host",
			mutate:  func(env map[string]string) { env["PROJECT_PROJECT_A_NEW_URL"] = "https://" },
			wantErr: "NEW_URL tidak valid",
		},
		{
			name:    "new url dengan kredensial",
			mutate:  func(env map[string]string) { env["PROJECT_PROJECT_A_NEW_URL"] = "https://user:pass@new-a.example" },
			wantErr: "NEW_URL tidak valid",
		},
		{
			name:    "new url dengan spasi",
			mutate:  func(env map[string]string) { env["PROJECT_PROJECT_A_NEW_URL"] = "https://new-a.example/a b" },
			wantErr: "NEW_URL tidak valid",
		},
		{
			name:    "app url tidak valid",
			mutate:  func(env map[string]string) { env["PROJECT_PROJECT_A_APP_URL"] = "data:text/html,halo" },
			wantErr: "APP_URL tidak valid",
		},
		{
			name:    "accent tidak valid",
			mutate:  func(env map[string]string) { env["PROJECT_PROJECT_A_ACCENT_COLOR"] = "red" },
			wantErr: "ACCENT_COLOR",
		},
		{
			name:    "slug duplikat",
			mutate:  func(env map[string]string) { env["PROJECTS"] = "project-a,project-a" },
			wantErr: "duplikat",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := validEnv()
			tt.mutate(env)
			_, err := loadWith(t, env)
			if err == nil {
				t.Fatalf("Load() seharusnya gagal")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want mengandung %q", err, tt.wantErr)
			}
		})
	}
}

func TestLoadRejectsUnsafeSlug(t *testing.T) {
	unsafe := []string{
		"Project-A",
		"project_a",
		"-project-a",
		"project-a-",
		"a b",
		"a/b",
		"pröject",
		"project--a/../b",
		strings.Repeat("a", 65),
	}

	for _, slug := range unsafe {
		t.Run(slug, func(t *testing.T) {
			cfg, err := loadWith(t, map[string]string{"PROJECTS": slug})
			if err == nil {
				t.Fatalf("Load() dengan slug %q seharusnya gagal, hasil = %+v", slug, cfg)
			}
			if !strings.Contains(err.Error(), "slug tidak valid") {
				t.Errorf("error = %q, want mengandung %q", err, "slug tidak valid")
			}
		})
	}
}

func TestReadableTextColor(t *testing.T) {
	tests := map[string]string{
		"#ffffff": "#0b0b0b",
		"#000000": "#ffffff",
		"#1d4ed8": "#ffffff",
		"#fde047": "#0b0b0b",
		"#0f766e": "#ffffff",
		"#777777": "#0b0b0b",
		"#FFF":    "#0b0b0b",
	}

	for color, want := range tests {
		if got := readableTextColor(color); got != want {
			t.Errorf("readableTextColor(%q) = %q, want %q", color, got, want)
		}
	}
}
