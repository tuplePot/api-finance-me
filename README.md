# Financial Tracker (Go)

Backend fitur **Finance** di josh-vault: transaksi, budget per kategori, dan
saving challenge. Database PostgreSQL di **Neon**.

## Menjalankan

```bash
cp .env.example .env      # isi DATABASE_URL, DATABASE_URL_UNPOOLED, API_KEY
go run ./cmd/migrate      # opsional — server juga migrasi otomatis saat start
go run ./cmd/api          # http://localhost:5080
```

Data contoh untuk dicoba (tanggal relatif terhadap hari ini):

```bash
go run ./cmd/seed         # isi / reset data contoh (tidak dobel)
go run ./cmd/seed -clear  # hapus HANYA data contoh, data buatanmu aman
```

Lalu di josh-vault (`.env.local`):

```env
VITE_FINANCE_API_URL="http://localhost:5080"
# VITE_API_KEY harus sama dengan API_KEY di backend ini
```

Tanpa `VITE_FINANCE_API_URL`, frontend memakai mock localStorage.

### Neon

Ambil dua connection string dari dashboard Neon (**Connect**):

| Variabel | Pakai yang mana | Untuk |
|---|---|---|
| `DATABASE_URL` | **Pooled** (host ada `-pooler`) | server |
| `DATABASE_URL_UNPOOLED` | **Direct** (tanpa `-pooler`) | migrasi |

Migrasi lewat PgBouncer (pooled) bisa gagal, makanya dipisah. Kalau
`DATABASE_URL_UNPOOLED` kosong, migrasi memakai `DATABASE_URL`.
Compute Neon bisa "tidur" (scale-to-zero); request pertama setelah idle bisa
lebih lambat beberapa detik.

### Environment

| Variabel | Default | Keterangan |
|---|---|---|
| `DATABASE_URL` | – | wajib |
| `DATABASE_URL_UNPOOLED` | `DATABASE_URL` | koneksi untuk migrasi |
| `API_KEY` | – | wajib; dicek dari header `X-API-Key` |
| `OWNER_ID` | `owner` | pemilik data untuk API key ini |
| `PORT` | `5080` | jangan 5060/5061 — diblokir browser (ERR_UNSAFE_PORT) |
| `TIMEZONE` | `Asia/Jakarta` | menentukan "hari ini" (minggu budget, deadline) |
| `CORS_ALLOWED_ORIGINS` | `*` | dipisah koma, mis. `http://localhost:5173` |
| `AUTO_MIGRATE` | `true` | migrasi saat server start |
| `LOG_LEVEL` / `LOG_FORMAT` | `info` / `text` | `json` untuk production |

## API

Semua route di bawah `/api` wajib header `X-API-Key`. Response:

```json
{ "success": true,  "message": "Budget created", "data": { ... } }
{ "success": false, "message": "Food & Drinks already has a budget", "code": "conflict", "fields": { ... } }
```

`code`: `validation_error` (422), `not_found` (404), `conflict` (409),
`unauthorized` (401), `internal_error` (500). `message` dalam bahasa Inggris
karena tampil langsung di UI.

| Method | Path | Body | Catatan |
|---|---|---|---|
| GET | `/health` | – | tanpa API key; cek koneksi DB |
| GET | `/api/transactions` | – | terbaru dulu |
| POST | `/api/transactions` | `{type, amount, category, date, note?}` | `date` = `yyyy-mm-dd` |
| PATCH | `/api/transactions/{id}` | `{type?, amount?, category?, date?, note?}` | field yang tidak dikirim tidak diubah; `note: ""` menghapus catatan |
| DELETE | `/api/transactions/{id}` | – | |
| GET | `/api/budgets` | – | |
| POST | `/api/budgets` | `{category, period, limit}` | `period` = `weekly` \| `monthly`; 1 budget per kategori |
| PATCH | `/api/budgets/{id}` | `{period, limit}` | kategori tidak bisa diganti |
| DELETE | `/api/budgets/{id}` | – | |
| GET | `/api/challenges` | – | challenge lewat deadline otomatis jadi `failed` |
| POST | `/api/challenges` | `{title, temptation{name,price}, goal{name,price}, targetAmount, deadline}` | |
| DELETE | `/api/challenges/{id}` | – | log ikut terhapus |
| POST | `/api/challenges/{id}/logs` | `{kind}` | `skip` \| `slip`; 409 jika sudah selesai |
| GET | `/api/habits` | – | urut waktu dibuat |
| POST | `/api/habits` | `{name, icon, cost, frequency, since}` | `frequency` = `daily` \| `weekly` \| `monthly`; `since` = `yyyy-mm-dd` |
| PATCH | `/api/habits/{id}` | `{name?, icon?, cost?, frequency?, since?}` | field yang tidak dikirim tidak diubah |
| DELETE | `/api/habits/{id}` | – | |
| GET | `/api/wishlist` | – | termurah dulu |
| POST | `/api/wishlist` | `{name, price}` | |
| DELETE | `/api/wishlist/{id}` | – | |

Uang = bigint Rupiah utuh (tidak pernah float).

### Aturan yang dijaga server

- **Skip** menambah `savedAmount` sebesar harga temptation; **slip** dicatat
  tapi tidak menambah apa-apa.
- Saat target tercapai → `success`. Baris challenge dikunci
  (`SELECT … FOR UPDATE`), jadi klik ganda / request bersamaan tidak saling
  menimpa `savedAmount` dan tidak bisa mencatat setelah challenge selesai.
- Habit ("habits I don't have") hanya menyimpan biaya per periode dan tanggal
  `since`. Uang yang tersimpan dihitung frontend (biaya per hari × jumlah hari
  sejak `since`), jadi tidak perlu job harian.
- Budget hanya menyimpan batas; pemakaian dihitung dari transaksi. Server
  tidak memblokir pengeluaran yang melebihi budget — frontend memperingatkan
  dulu dan user bisa memilih "Save anyway".

## Struktur

```
cmd/api          composition root (repository → usecase → handler → router)
cmd/migrate      migrasi manual
cmd/seed         data contoh
migrations/      skema SQL (di-embed ke binary)
internal/
  transaction/   pemasukan / pengeluaran
  budget/        batas pengeluaran per kategori (weekly / monthly)
  challenge/     saving challenge + log skip/slip
  habit/         habit yang tidak kamu punya (rokok, vape, jajan)
  wishlist/      barang incaran — satuan konversi uang tersimpan
  shared/        apperror, validator, clock (zona waktu), owner (context)
platform/        config, logger, postgres, httpserver
```

Setiap modul: `domain` (aturan murni) → `usecase` (alur, mendefinisikan
interface repository) → `handler` (HTTP ⇄ usecase) → `repository` (Postgres).

Auth saat ini: satu API key = satu pemilik. Semua tabel sudah punya
`owner_id`, jadi pindah ke login multi-user cukup mengganti middleware
`httpserver.APIKey`.

## Test

```bash
go test ./...
```
