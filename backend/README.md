# backend

Kontrata Go servisi. GraphQL API (Chi router, gqlgen) ve iş kuralları burada yaşar.

Go modül yolu: `github.com/oz-fatma/kontrata/backend`.
İç paketler `internal/` altında durur.

## Gereksinimler

- Go 1.25 veya üzeri
- MongoDB 8 (yerel geliştirmede `docker compose up -d`)

## Nasıl çalıştırılır

Ortam değişkenleri için `.env.example` dosyasına bakın. `backend/.env` varsa açılışta yüklenir; yoksa ortam değişkenleri olduğu gibi kullanılır.

```sh
make run
```

Derleme (`bin/api` üretir; sürüm ldflags ile gömülür):

```sh
make build
```

Sürümü açık vermek için: `make build VERSION=0.1.0`

## Uçlar

| Metot | Yol | Açıklama |
| --- | --- | --- |
| `GET` | `/healthz` | Sağlık denetimi. Veritabanı bağlıysa `200` ve `"database":"connected"`; erişilemezse `503` ve `"database":"unreachable"`. |
| `POST`/`GET` | `/graphql` | GraphQL API. |
| `GET` | `/playground` | GraphQL Playground. Yalnızca `GRAPHQL_PLAYGROUND=true` iken açık. |

Şema `graph/schema.graphqls` dosyasındadır; Go modelleri ve resolver iskeleti `make generate` ile üretilir.

```sh
make generate
```

## Ortam değişkenleri

| Değişken | Zorunlu | Varsayılan | Açıklama |
| --- | --- | --- | --- |
| `PORT` | hayır | `8080` | HTTP dinleme kapısı |
| `MONGO_URI` | evet | — | MongoDB bağlantı adresi. Günlüğe yazılmaz. |
| `GRAPHQL_PLAYGROUND` | hayır | kapalı | `true` ise `/playground` açılır. |
