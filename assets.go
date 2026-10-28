// Package assets menyimpan template HTML dan aset statis yang di-embed
// ke dalam binary sehingga image runtime tidak perlu menyalin direktori.
package assets

import "embed"

// Templates berisi seluruh template halaman.
//
//go:embed templates/*.html
var Templates embed.FS

// Static berisi CSS dan favicon.
//
//go:embed static
var Static embed.FS
