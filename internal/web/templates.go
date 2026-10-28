package web

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"html/template"
	"io/fs"

	"gantibdomain"
)

type templates struct {
	index     *template.Template
	migration *template.Template
	notFound  *template.Template
}

func parseTemplates() (*templates, error) {
	set, err := template.ParseFS(assets.Templates, "templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("gagal memuat template: %w", err)
	}

	lookup := func(name string) (*template.Template, error) {
		t := set.Lookup(name)
		if t == nil {
			return nil, fmt.Errorf("template %q tidak ditemukan", name)
		}
		return t, nil
	}

	index, err := lookup("index.html")
	if err != nil {
		return nil, err
	}
	migration, err := lookup("migration.html")
	if err != nil {
		return nil, err
	}
	notFound, err := lookup("404.html")
	if err != nil {
		return nil, err
	}

	return &templates{index: index, migration: migration, notFound: notFound}, nil
}

// assetVersion menghitung hash pendek dari isi aset untuk cache busting.
func assetVersion(fsys fs.FS, name string) (string, error) {
	data, err := fs.ReadFile(fsys, name)
	if err != nil {
		return "", fmt.Errorf("gagal membaca aset %q: %w", name, err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])[:12], nil
}
