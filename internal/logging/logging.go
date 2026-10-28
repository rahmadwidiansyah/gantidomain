// Package logging menyiapkan logger JSON untuk stdout dan, jika diminta,
// untuk file di direktori yang dapat ditulis oleh proses.
package logging

import (
	"fmt"
	"io"
	"log/slog"
	"os"
)

// New membuat logger JSON. Jika path kosong, log hanya ditulis ke stdout.
// Jika path diisi, log ditulis ke stdout dan file (mode append, 0644).
// Pemanggil bertanggung jawab memanggil fungsi close yang dikembalikan.
func New(path string) (*slog.Logger, func() error, error) {
	writer := io.Writer(os.Stdout)
	closeFn := func() error { return nil }

	if path != "" {
		file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			return nil, nil, fmt.Errorf("gagal membuka LOG_FILE %q: %w", path, err)
		}
		writer = io.MultiWriter(os.Stdout, file)
		closeFn = file.Close
	}

	logger := slog.New(slog.NewJSONHandler(writer, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	return logger, closeFn, nil
}
