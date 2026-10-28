# Ganti Domain — Landing Page Pemberitahuan Perpindahan Domain

Web ringan untuk memberi tahu pengunjung bahwa sebuah project telah berpindah ke domain baru. Satu instalasi dapat melayani banyak project, masing-masing dengan halaman sendiri di `/{slug}`.

Dibangun dengan Go standard library, template HTML yang di-embed ke binary, dan Nginx sebagai reverse proxy. Tanpa database, tanpa panel admin, dan tanpa JavaScript pada jalur render utama.

## Fitur

- Banyak project dalam satu layanan, dipilih berdasarkan path `/{slug}`.
- Konfigurasi sepenuhnya melalui environment variable (`.env`).
- Tombol **Buka Website Baru** sebagai CTA utama.
- Tombol **Download Aplikasi Baru** hanya muncul jika `APP_URL` diisi.
- Halaman indeks, halaman 404, dan redirect 301 untuk trailing slash.
- Health check di `/healthz`.
- Validasi konfigurasi saat startup: slug, URL, dan warna aksen.
- Escaping otomatis lewat `html/template`, CSP dengan nonce, dan header keamanan dasar.
- Container aplikasi non-root, read-only, dan tanpa port yang dipublikasikan langsung.
- Image siap pakai di GHCR, dibangun otomatis oleh GitHub Actions untuk amd64 dan arm64.

## Arsitektur

```
Browser
   │  http://host:4114
   ▼
Nginx (container, port 80)  ──proxy──▶  Go app (container, port 8080)
   │                                        │
   ├─ ./nginx/default.conf (bind mount)     ├─ template & CSS di-embed di binary
```

- Hanya Nginx yang memublikasikan port host (`4114`).
- Aplikasi Go berada di network bridge biasa.
- Tidak ada database, Redis, atau named volume Docker.

## Persyaratan

- Docker Engine 24+ dan Docker Compose v2.
- Port `4114` tersedia di host.
- Untuk build lokal: Go 1.27+.

Image publik: `ghcr.io/rahmadwidiansyah/gantidomain` (tidak perlu login untuk pull).

## Quick start (image GHCR)

```bash
git clone https://github.com/rahmadwidiansyah/gantidomain.git
cd gantidomain

cp .env.example .env
# Edit .env dan ganti domain contoh dengan domain asli.

docker compose pull
docker compose up -d
docker compose ps
```

Buka `http://localhost:4114` atau `http://IP-SERVER:4114`.

Verifikasi cepat:

```bash
curl -i http://localhost:4114/healthz      # 200 OK
curl -i http://localhost:4114/project-a    # halaman project-a
```

## Konfigurasi `.env`

```dotenv
IMAGE_TAG=latest
PORT=8080
PROJECTS=project-a,project-b

PROJECT_PROJECT_A_NAME=Project A
PROJECT_PROJECT_A_OLD_DOMAIN=old-a.example
PROJECT_PROJECT_A_NEW_URL=https://new-a.example
PROJECT_PROJECT_A_APP_URL=https://downloads.example/app-a
PROJECT_PROJECT_A_DESCRIPTION=Website resmi Project A telah berpindah ke alamat baru.
PROJECT_PROJECT_A_MESSAGE=Gunakan alamat baru untuk mengakses layanan.
PROJECT_PROJECT_A_ACCENT_COLOR=#1d4ed8
```

### Variabel global

| Variabel | Wajib | Default | Keterangan |
| --- | --- | --- | --- |
| `IMAGE_TAG` | tidak | `latest` | Tag image GHCR. Pin ke `sha-<commit>` atau tag versi untuk deployment reproducible. |
| `PORT` | tidak | `8080` | Port internal aplikasi Go. Tidak dipublikasikan ke host. |
| `PROJECTS` | ya | — | Daftar slug dipisahkan koma; urutannya dipakai di halaman indeks. Boleh kosong. |

### Variabel per project

Slug `project-a` dipetakan ke prefix `PROJECT_PROJECT_A_*` (huruf besar, `-` menjadi `_`).

| Variabel | Wajib | Keterangan |
| --- | --- | --- |
| `PROJECT_<SLUG>_NAME` | ya | Nama project yang ditampilkan. |
| `PROJECT_<SLUG>_NEW_URL` | ya | URL website baru; hanya `http`/`https`, wajib punya host, tanpa kredensial. |
| `PROJECT_<SLUG>_APP_URL` | tidak | URL unduhan aplikasi. Tombol download tidak dirender jika kosong. |
| `PROJECT_<SLUG>_OLD_DOMAIN` | tidak | Domain lama; ditampilkan sebagai teks, bukan tautan. |
| `PROJECT_<SLUG>_DESCRIPTION` | tidak | Deskripsi singkat. |
| `PROJECT_<SLUG>_MESSAGE` | tidak | Catatan migrasi tambahan. |
| `PROJECT_<SLUG>_ACCENT_COLOR` | tidak | Warna aksen `#RGB` atau `#RRGGBB`; default `#1d4ed8`. |

Aturan validasi:

- Slug: huruf kecil `a-z`, angka `0-9`, tanda hubung; maksimal 64 karakter; tidak diawali/diakhiri tanda hubung; tidak boleh duplikat.
- URL: skema `http`/`https`, host tidak kosong, maksimal 2048 karakter, tanpa spasi/karakter kontrol, tanpa `user:pass@`.
- `ACCENT_COLOR` divalidasi sebagai hex; warna teks tombol dihitung otomatis agar kontras memenuhi WCAG AA.
- Konfigurasi tidak valid membuat proses gagal saat startup dengan pesan yang menyebut variabel bermasalah.

### Menambah project

```dotenv
PROJECTS=project-a,project-b,project-c

PROJECT_PROJECT_C_NAME=Project C
PROJECT_PROJECT_C_OLD_DOMAIN=old-c.example
PROJECT_PROJECT_C_NEW_URL=https://new-c.example
PROJECT_PROJECT_C_APP_URL=
PROJECT_PROJECT_C_DESCRIPTION=Website Project C telah berpindah.
PROJECT_PROJECT_C_ACCENT_COLOR=#7c3aed
```

Terapkan perubahan konfigurasi:

```bash
docker compose up -d --force-recreate
```

Tidak perlu rebuild image karena konfigurasi dibaca saat container dibuat.

## Routing

| Path | Perilaku |
| --- | --- |
| `/` | Halaman indeks berisi daftar project |
| `/{slug}` | Halaman migrasi project |
| `/{slug}/` | `301` ke `/{slug}`; query string dipertahankan |
| `/healthz` | `200 OK` selama proses berjalan |
| `/static/...` | CSS dan favicon (di-embed di binary) |
| Path tidak dikenal | `404` dengan halaman HTML |
| Metode selain `GET`/`HEAD` | `405` + header `Allow: GET, HEAD` |

Tidak ada redirect otomatis ke domain baru. Pengunjung selalu melihat halaman pemberitahuan dengan tombol menuju alamat baru.

## Operasi

```bash
# Lihat status dan kesehatan container
docker compose ps

# Ambil image terbaru lalu jalankan
docker compose pull
docker compose up -d

# Terapkan perubahan .env
docker compose up -d --force-recreate

# Ganti tag image tanpa mengedit .env
IMAGE_TAG=sha-d61814a docker compose up -d

# Log stdout container
docker compose logs -f --tail=100

# Hentikan layanan
docker compose down
```

`docker compose down` tidak menghapus `.env` maupun konfigurasi Nginx.

## Build lokal

File utama memakai image GHCR. Untuk mengembangkan kode secara lokal, gunakan override build:

```bash
docker compose -f docker-compose.yml -f docker-compose.build.yml up -d --build
```

Perintah tersebut membangun image dari `Dockerfile` dan memberinya tag sesuai `IMAGE_TAG`.

Jalankan test tanpa Docker:

```bash
go vet ./...
go test ./...
```

## CI/CD

Workflow `.github/workflows/docker.yml` berjalan pada:

- push ke `main`
- tag `v*`
- pull request ke `main` (build saja, tanpa push)
- `workflow_dispatch`

Job `test` menjalankan `go vet` dan `go test`. Job `image` membangun `linux/amd64` dan `linux/arm64`, lalu push ke GHCR:

```
ghcr.io/rahmadwidiansyah/gantidomain:latest   # default branch
ghcr.io/rahmadwidiansyah/gantidomain:main
ghcr.io/rahmadwidiansyah/gantidomain:sha-<short>
ghcr.io/rahmadwidiansyah/gantidomain:1.2.3    # dari tag v1.2.3
ghcr.io/rahmadwidiansyah/gantidomain:1.2
```

Tidak ada secret tambahan; autentikasi memakai `GITHUB_TOKEN` dengan permission `packages: write`. Image menyertakan SBOM dan provenance.

Jika package diubah menjadi private, server perlu login sebelum pull:

```bash
echo "$GHCR_TOKEN" | docker login ghcr.io -u <username> --password-stdin
```

## Struktur repository

```
.
├── .github/workflows/docker.yml
├── cmd/server/main.go
├── internal/
│   ├── config/            # loader & validasi environment
│   └── web/               # routing, header, render template
├── templates/             # index, migration, 404
├── static/                # css, favicon
├── nginx/default.conf
├── Dockerfile
├── docker-compose.yml
├── docker-compose.build.yml
├── .env.example
└── PLAN.md
```

Template dan aset statis di-embed ke binary dengan `//go:embed`.

## Keamanan

- Seluruh nilai konfigurasi di-escape oleh `html/template`; tidak ada HTML yang disusun dari string mentah.
- URL hanya menerima `http`/`https` dan divalidasi saat startup.
- CSP: `default-src 'none'`, `style-src 'self' 'nonce-...'`, `frame-ancestors 'none'`.
- Header: `X-Content-Type-Options`, `Referrer-Policy`, `X-Frame-Options`, `Permissions-Policy`.
- Container aplikasi berjalan sebagai UID 10001, root filesystem read-only, `cap_drop: ALL`, `no-new-privileges`.
- Tidak ada directory listing; `.env` tidak pernah disajikan dan tidak masuk ke image.
- Request dibatasi oleh Nginx (`client_max_body_size 8k`) dan timeout server Go.
- Tidak ada endpoint yang dapat mengubah konfigurasi melalui HTTP.

## Troubleshooting

**Halaman 404 padahal slug sudah diisi**
Pastikan slug ada di `PROJECTS` dan variabel memakai prefix yang benar. Contoh `project-a` → `PROJECT_PROJECT_A_*`, bukan `PROJECT_A_*`. Cek `docker compose logs web` untuk pesan validasi.

**Container `web` langsung berhenti**
Lihat `docker compose logs web`. Penyebab umum: `NEW_URL` kosong/tidak valid, `ACCENT_COLOR` salah format.

**502 Bad Gateway dari Nginx**
Container `web` belum sehat. Cek `docker compose ps` dan `docker compose logs web`; `depends_on` hanya menunda start, bukan menjamin aplikasi tetap hidup.

**Port 4114 sudah dipakai**
Ubah pemetaan port di `docker-compose.yml` (misalnya `"4115:80"`) atau hentikan layanan yang memakai port tersebut.

**Image `latest` tidak berubah setelah update**
Jalankan `docker compose pull` sebelum `docker compose up -d`. Untuk reproducibility, pin `IMAGE_TAG` ke `sha-<commit>` atau tag versi.
