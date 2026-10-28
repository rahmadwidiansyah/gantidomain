# PLAN.md — Multi-Project Domain Migration Landing Pages

## 1. Tujuan

Membangun web ringan untuk menampilkan pemberitahuan bahwa domain sebuah project telah berpindah ke alamat baru. Setiap project memiliki halaman sendiri berdasarkan path, misalnya:

- `https://domain-lama.example/project-a`
- `https://domain-lama.example/project-b`

Setiap halaman menampilkan:

- Nama project dan pesan bahwa domain telah berpindah.
- Tombol utama **Buka Website Baru**.
- Tombol **Download Aplikasi Baru** jika tautan aplikasi tersedia.
- Informasi opsional seperti domain lama, domain baru, deskripsi singkat, dan catatan migrasi.

Tidak ada login, database, panel admin, atau JavaScript berat. Seluruh konfigurasi project disimpan melalui environment variable pada `.env`.

## 2. Keputusan teknologi

- Bahasa/runtime aplikasi: **Go**.
- HTTP server: `net/http` bawaan Go.
- Templating: `html/template` bawaan Go.
- Frontend: HTML semantik + CSS responsif, tanpa framework JavaScript.
- Reverse proxy: **Nginx**.
- Deployment: **Docker Compose**.
- Port publik host: **4114**, diteruskan ke Nginx di container.
- Konfigurasi: `.env` untuk daftar project dan detail tiap project.
- Penyimpanan: tidak ada database maupun volume data persisten.

Alasan: Go menghasilkan satu binary, startup cepat, penggunaan memori relatif kecil, dan tidak membutuhkan runtime Node.js di production. Nginx menangani koneksi HTTP di depan aplikasi.

## 3. Fitur wajib

### 3.1 Routing multi-project

- Path `/{project-slug}` memilih konfigurasi project yang sesuai.
- Slug hanya boleh berupa huruf kecil `a-z`, angka `0-9`, dan tanda hubung, tidak diawali/diakhiri tanda hubung, maksimal 64 karakter.
- Slug yang tidak terdaftar mengembalikan HTTP 404.
- `/healthz` menyediakan health check.
- `/` menampilkan halaman indeks sederhana berisi project yang dikonfigurasi, atau halaman petunjuk singkat. Request yang tidak dikenal tidak diarahkan ke salah satu project.
- Normalisasi trailing slash: `/project-a/` diarahkan ke `/project-a` dengan HTTP 301, query string dipertahankan.
- Tidak ada redirect otomatis ke domain baru; halaman pemberitahuan ditampilkan dengan tombol yang jelas.

### 3.2 Data per project

Setiap project mendukung:

- `name`: nama yang ditampilkan (wajib).
- `old_domain`: domain lama, opsional (hanya ditampilkan sebagai teks).
- `new_url`: URL tujuan website baru, wajib.
- `app_url`: URL unduhan aplikasi, opsional.
- `description`: keterangan singkat, opsional.
- `message`: pesan migrasi, opsional.
- `accent_color`: warna aksen, opsional, hanya menerima `#RGB` atau `#RRGGBB`.

Jika `app_url` kosong, tombol download tidak dirender.

### 3.3 Desain UI/UX

- Bahasa Indonesia.
- Mobile-first, nyaman di layar ponsel dan desktop.
- Tampilan profesional, bersih, tanpa emoji dekoratif.
- Hierarki jelas: status migrasi, nama project, penjelasan, tombol website baru, lalu tombol download.
- Tombol website baru menjadi CTA utama; tombol download menjadi CTA sekunder.
- Kontras warna memenuhi WCAG AA; warna teks tombol aksen dihitung otomatis dari luminance.
- Fokus keyboard terlihat; semua tombol berupa tautan HTML asli.
- `rel="noopener noreferrer"` pada tautan eksternal `target="_blank"`.
- Tanpa font eksternal, CDN, analytics, atau library frontend.
- CSS kecil disimpan lokal; mendukung `prefers-reduced-motion`.
- Metadata title, description, theme-color, dan favicon sederhana.

### 3.4 Keamanan dan ketahanan

- Semua nilai konfigurasi di-escape oleh `html/template`.
- `new_url` dan `app_url` divalidasi: hanya `http`/`https`, wajib memiliki host, tanpa kredensial, tanpa karakter kontrol.
- Slug divalidasi sebelum digunakan sebagai key routing.
- HTML tidak disusun dari string konfigurasi mentah.
- Tidak ada directory listing; `.env` tidak pernah disajikan.
- Nginx hanya meneruskan request yang diperlukan ke aplikasi.
- Header keamanan: `X-Content-Type-Options: nosniff`, `Referrer-Policy: strict-origin-when-cross-origin`, `X-Frame-Options: DENY`, dan CSP dengan nonce untuk blok `<style>` aksen.
- Ukuran request dibatasi dan server Go memakai timeout.
- Container aplikasi berjalan sebagai user non-root (UID 10001), read-only, `cap_drop: ALL`.
- Tidak ada endpoint yang dapat mengubah konfigurasi melalui HTTP.
- Log tidak mencatat secret.

## 4. Struktur repository

```
domain-migration/
├── .github/
│   └── workflows/
│       └── docker.yml
├── README.md
├── PLAN.md
├── Dockerfile
├── docker-compose.yml
├── docker-compose.build.yml
├── .env
├── .env.example
├── .dockerignore
├── .gitignore
├── go.mod
├── assets.go                  # //go:embed template dan static
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── config/
│   │   ├── config.go
│   │   └── config_test.go
│   └── web/
│       ├── handler.go
│       ├── templates.go
│       └── handler_test.go
├── templates/
│   ├── index.html
│   ├── migration.html
│   └── 404.html
├── static/
│   ├── css/
│   │   └── style.css
│   └── favicon.svg
└── nginx/
    └── default.conf
```

Template dan aset statis di-embed dengan `//go:embed` sehingga image runtime hanya berisi binary. Tidak ada `go.sum` karena hanya memakai standard library.

## 5. Kontrak konfigurasi .env

Gunakan satu daftar slug yang urut, lalu variabel per slug.

```dotenv
IMAGE_TAG=latest
PORT=8080
PROJECTS=project-a,project-b

PROJECT_PROJECT_A_NAME=Project A
PROJECT_PROJECT_A_OLD_DOMAIN=old-a.example
PROJECT_PROJECT_A_NEW_URL=https://new-a.example
PROJECT_PROJECT_A_APP_URL=https://downloads.example/app-a
PROJECT_PROJECT_A_DESCRIPTION=Website resmi Project A telah berpindah.
PROJECT_PROJECT_A_MESSAGE=Gunakan alamat baru untuk mengakses layanan.
PROJECT_PROJECT_A_ACCENT_COLOR=#1d4ed8

PROJECT_PROJECT_B_NAME=Project B
PROJECT_PROJECT_B_OLD_DOMAIN=old-b.example
PROJECT_PROJECT_B_NEW_URL=https://new-b.example
PROJECT_PROJECT_B_APP_URL=
PROJECT_PROJECT_B_DESCRIPTION=Alamat resmi Project B telah diperbarui.
PROJECT_PROJECT_B_MESSAGE=Bookmark alamat baru agar mudah ditemukan.
```

Aturan konfigurasi:

- `PROJECTS` menentukan project yang dimuat; urutannya dipakai pada halaman indeks.
- Slug `project-a` dipetakan ke prefix `PROJECT_PROJECT_A_*` (huruf besar, `-` menjadi `_`).
- `NAME` dan `NEW_URL` wajib.
- `APP_URL`, `OLD_DOMAIN`, `DESCRIPTION`, `MESSAGE`, dan `ACCENT_COLOR` opsional.
- Aplikasi memvalidasi konfigurasi saat startup. Jika ada project tidak valid, startup gagal dengan pesan yang jelas.
- `.env` dimasukkan ke container oleh Docker Compose melalui `env_file`, bukan dibaca manual oleh aplikasi.
- `.env` dikecualikan dari image dan git.

## 6. Docker dan Nginx

### 6.1 Docker Compose

- Service `web`: image `ghcr.io/rahmadwidiansyah/gantidomain:${IMAGE_TAG:-latest}`, listen pada `0.0.0.0:8080` di jaringan internal Compose, tanpa publish port.
- Service `nginx`: reverse proxy, publish `4114:80`.
- Nginx menjadi satu-satunya service yang mempublikasikan port host.
- Network internal khusus untuk komunikasi Nginx ke aplikasi.
- Health check aplikasi via `wget`, `depends_on` dengan `condition: service_healthy`.
- `restart: unless-stopped`, resource limit memori 128 MB per service.
- Aplikasi: `read_only`, `cap_drop: ALL`, `no-new-privileges`, `tmpfs` kecil untuk `/tmp`.
- Tidak ada database, Redis, atau volume Docker.
- Build local memakai override `docker-compose.build.yml`; build multi-stage: `golang:1.27.1-alpine` untuk build, `alpine:3.22` untuk runtime.
- Base image di-pin, bukan `latest`; tag image aplikasi dapat dipin melalui `IMAGE_TAG`.

### 6.2 Nginx

- Listen pada port 80 di container.
- Proxy ke `web:8080` dengan `Host`, `X-Real-IP`, `X-Forwarded-For`, `X-Forwarded-Proto`.
- `client_max_body_size 8k` karena tidak ada upload.
- Timeout koneksi/read/send yang wajar.
- gzip untuk HTML, CSS, SVG, dan teks.
- Static assets tetap dilayani oleh Go (di-embed di binary).
- Tidak ada catch-all redirect ke domain project mana pun, tidak ada `autoindex`.

## 7. Performa

- Tanpa framework frontend dan tanpa JavaScript pada jalur render utama.
- Tanpa query database atau request API eksternal.
- Konfigurasi dimuat sekali saat startup dan disimpan dalam map read-only.
- Template dan CSS di-embed dalam binary.
- Cache: HTML `no-cache`, 404/redirect `no-store`, aset statis `immutable` dengan query `?v=<hash>`.
- Tidak ada animasi berat atau gambar besar.
- Halaman tetap berfungsi ketika JavaScript dimatikan.

## 8. HTTP status dan routing

| Path | Perilaku |
| --- | --- |
| `/` | Halaman indeks project atau halaman petunjuk |
| `/{slug}` | Halaman migrasi untuk slug yang terdaftar |
| `/{slug}/` | Redirect 301 ke `/{slug}` (query dipertahankan) |
| `/healthz` | `200 OK` jika proses sehat |
| `/static/...` | CSS dan favicon dari embed FS |
| Path tidak terdaftar | `404 Not Found` |
| Metode selain `GET`/`HEAD` | `405 Method Not Allowed` + `Allow: GET, HEAD` |

## 9. Contoh tampilan halaman

- Label status: **Alamat website telah diperbarui**
- Judul: **Project A kini memiliki alamat baru**
- Deskripsi: "Kami telah memindahkan website ke alamat baru. Silakan gunakan tautan resmi di bawah."
- Tombol utama: **Buka Website Baru**
- Tombol sekunder (jika `app_url` ada): **Download Aplikasi Baru**
- Informasi tambahan: "Domain sebelumnya: old-a.example" (bukan tautan)
- Footer: "Pastikan Anda menggunakan alamat resmi yang tercantum di halaman ini."

Domain lama tidak ditampilkan sebagai tautan. Aplikasi tidak diklaim tersedia jika `app_url` tidak diatur.

## 10. Pengujian wajib

Konfigurasi:

- Startup berhasil dengan dua project.
- Startup gagal dengan pesan jelas jika `NAME` atau `NEW_URL` kosong/tidak valid.
- Project tanpa `APP_URL` tampil dengan satu tombol.
- Slug duplikat ditolak.
- Slug yang tidak aman ditolak.
- `javascript:` dan skema selain HTTP/HTTPS ditolak.
- `ACCENT_COLOR` tidak valid ditolak.

Routing:

- `/project-a` dan `/project-b` menampilkan data yang benar dan tidak tertukar.
- `/unknown-project` mengembalikan 404.
- Trailing slash dialihkan sesuai spesifikasi.
- `/healthz` mengembalikan 200.
- Metode selain GET/HEAD ditolak dengan 405 dan header `Allow`.

Keamanan:

- Nilai konfigurasi dengan karakter HTML tampil sebagai teks aman.
- `.env` tidak dapat diakses dari browser.
- Directory listing tidak aktif.
- Header keamanan dasar hadir dan nonce CSP cocok dengan blok style.
- Container aplikasi berjalan non-root.

Docker:

- `docker compose config` berhasil.
- `docker compose up -d --build` berhasil.
- `curl -i http://localhost:4114/healthz` menghasilkan 200.
- `curl -i http://localhost:4114/project-a` menampilkan halaman yang benar.
- Restart/recreate container tidak menghilangkan konfigurasi dari `.env`.

Semua pengujian di atas dicakup oleh `go test ./...` (unit test config dan web) serta verifikasi Docker/curl manual.

## 11. Perintah operasi

```bash
# Validasi konfigurasi Compose
docker compose config

# Build dan jalankan
docker compose up -d --build

# Lihat status
docker compose ps

# Lihat log
docker compose logs -f --tail=100

# Restart setelah mengubah .env
docker compose up -d --build --force-recreate

# Hentikan layanan
docker compose down
```

Akses melalui `http://localhost:4114`. Jika dipasang di belakang reverse proxy lain, proxy luar diarahkan ke port host 4114.

## 12. Kriteria selesai

- Repository memiliki seluruh file yang diperlukan untuk build dan menjalankan layanan.
- `docker-compose.yml` memublikasikan layanan pada port 4114.
- Nginx menjadi reverse proxy di depan aplikasi Go.
- Project dapat ditambah atau diubah cukup melalui `.env`, lalu recreate service.
- Setiap `/slug` menampilkan nama, pesan, URL website, dan URL aplikasi sesuai project.
- Tombol aplikasi tersembunyi jika URL tidak tersedia.
- Tidak ada autentikasi, database, atau dependensi frontend berat.
- Routing, validasi URL, halaman 404, health check, dan pemeriksaan Docker lulus.

## 13. Di luar cakupan

- Panel admin atau login.
- Database.
- Statistik pengunjung dan analytics.
- Pengelolaan konfigurasi melalui UI.
- Pemendek URL.
- Redirect otomatis permanen.
- Pengunduhan file aplikasi yang di-host langsung oleh layanan ini.
- TLS/HTTPS yang dikelola langsung oleh container ini. TLS dapat ditangani reverse proxy di depan layanan.

## 14. Keputusan implementasi

- **CSP + accent color**: warna aksen disuntik melalui `<style nonce="...">` per request; CSP memakai `style-src 'self' 'nonce-...'` sehingga tidak perlu `unsafe-inline`.
- **Kontras tombol aksen**: warna teks hitam/putih dipilih otomatis dari kontras tertinggi, sehingga selalu memenuhi ambang WCAG AA untuk teks normal.
- **HEAD**: template dirender ke buffer, `Content-Length` dihitung, body hanya dikirim untuk GET.
- **Runtime**: `alpine:3.22` + user 10001, healthcheck `wget` pada `/healthz`.
- **Aset**: `assets.go` di root meng-embed `templates/` dan `static/`; tanpa `go.sum` karena hanya stdlib.
- **Keamanan tambahan**: validasi URL menolak kredensial, CSP `default-src 'none'`, `Permissions-Policy`, container read-only, dan directory listing diblokir eksplisit pada route `/static/`.

## 15. CI (GitHub Actions)

Workflow `.github/workflows/docker.yml`:

- Trigger: push ke `main`, tag `v*`, pull request ke `main`, dan `workflow_dispatch`.
- Job `test` menjalankan `go vet ./...` dan `go test ./...` sebelum image dibangun.
- Job `image` membangun image `linux/amd64` dan `linux/arm64`, lalu push ke `ghcr.io/<owner>/<repo>`.
- Push hanya dilakukan untuk non-PR; PR hanya membangun image tanpa publikasi.
- Tag image: `latest` (default branch), nama branch, `sha-<short>`, serta tag semver (`1.2.3`, `1.2`) dari tag `v*`.
- Autentikasi memakai `GITHUB_TOKEN` dengan permission `packages: write`; tidak perlu secret tambahan.
- Cache build memakai GitHub Actions cache; image menyertakan SBOM dan provenance.
- Package GHCR pertama kali dibuat otomatis dan defaultnya private. Ubah visibility menjadi public di pengaturan package jika image perlu ditarik tanpa login.

