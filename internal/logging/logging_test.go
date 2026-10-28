package logging

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewWithoutFile(t *testing.T) {
	logger, closeFn, err := New("")
	if err != nil {
		t.Fatalf("New(\"\") error = %v", err)
	}
	logger.Info("tanpa file")
	if err := closeFn(); err != nil {
		t.Errorf("close error = %v", err)
	}
}

func TestNewWritesJSONToFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "server.log")

	logger, closeFn, err := New(path)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	logger.Info("pesan uji", "project", "project-a")
	if err := closeFn(); err != nil {
		t.Fatalf("close error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	content := string(data)
	for _, want := range []string{`"level":"INFO"`, `"msg":"pesan uji"`, `"project":"project-a"`} {
		if !strings.Contains(content, want) {
			t.Errorf("file log tidak mengandung %q:\n%s", want, content)
		}
	}
	if !strings.HasSuffix(content, "\n") {
		t.Error("file log harus berakhir dengan baris baru")
	}
}

func TestNewAppendsToExistingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "server.log")
	if err := os.WriteFile(path, []byte("baris lama\n"), 0o640); err != nil {
		t.Fatal(err)
	}

	logger, closeFn, err := New(path)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	logger.Info("baris baru")
	if err := closeFn(); err != nil {
		t.Fatalf("close error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(data), "baris lama\n") {
		t.Errorf("isi lama hilang:\n%s", data)
	}
	if !strings.Contains(string(data), "baris baru") {
		t.Errorf("isi baru tidak ditulis:\n%s", data)
	}
}

func TestNewInvalidPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tidak-ada", "server.log")
	if _, _, err := New(path); err == nil {
		t.Fatal("New() dengan direktori tidak ada seharusnya gagal")
	}
}
