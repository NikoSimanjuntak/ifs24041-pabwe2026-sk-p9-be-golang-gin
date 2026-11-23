# Delcom Blog — Backend (Golang Gin, port 8081)

Clean Architecture + Feature-First: `internal/features/{auth,post,ai}`.

## Menjalankan
```bash
psql -U postgres -c "CREATE DATABASE pabwe_p9_blog;"
go mod tidy                 # mengunduh dependensi & membuat go.sum
go run cmd/server/main.go   # http://localhost:8081  | Swagger: /swagger/index.html
```
Akun admin dibuat otomatis (seed) dari `ADMIN_USERNAME` / `ADMIN_PASSWORD` pada `.env`.

## Pengujian
```bash
go test -v -cover ./internal/...
```
Test memakai SQLite in-memory (tanpa PostgreSQL) dan `httptest` untuk Search Engine & OpenAI.

## Build
```bash
go build -o server.exe cmd/server/main.go   # .env harus berada di samping binary
```

## Catatan integrasi Search Engine
`internal/features/ai/search.go` memanggil `GET {SEARCH_ENGINE_ENDPOINT}?q=...` dan membaca
`{"results":[...]}`, `{"data":[...]}`, atau `{"items":[...]}` (item: title, url/link, snippet/content/description).
Jika format API delcom.org berbeda, cukup sesuaikan struct `rawSearchResponse`/`rawItem` di berkas itu.
`OPENAI_ENDPOINT` dipanggil sebagai `POST {endpoint}/chat/completions` (format OpenAI-compatible).
