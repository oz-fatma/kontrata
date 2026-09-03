# backend

Kontrata Go servisi. GraphQL API (Chi router, gqlgen) ve iş kuralları burada yaşar.

Go modül yolu: `github.com/oz-fatma/kontrata/backend`.
İç paketler `internal/` altında durur.

## Gereksinimler

- Go 1.25 veya üzeri
- MongoDB 8, tek düğümlü replica set (`rs0`; yerel geliştirmede `docker compose up -d`)

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

Electron paketlemesi platform başına ayrı ikili ister:

```sh
make build-darwin    # bin/api-darwin-arm64, bin/api-darwin-amd64
make build-windows   # bin/api-windows-amd64.exe
```

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

CI'a göndermeden önce `make verify` çalıştırın. `go mod tidy` sonrası `go.mod` veya `go.sum` değişmişse komut hata verir; dosyaları commit etmek gerekir. Ardından `go vet` ve `go test` çalışır.

```sh
make verify
```

## Ortam değişkenleri

| Değişken | Zorunlu | Varsayılan | Açıklama |
| --- | --- | --- | --- |
| `PORT` | hayır | `8080` | HTTP dinleme kapısı |
| `MONGO_URI` | evet | — | MongoDB bağlantı adresi. Günlüğe yazılmaz. |
| `MONGO_DATABASE` | hayır | `kontrata` | Veritabanı adı. Testler `kontrata_test_*` kullanır. |
| `GRAPHQL_PLAYGROUND` | hayır | kapalı | `true` ise `/playground` açılır. |
| `MAILER` | hayır | `console` | `console` veya `smtp`. Konsol alıcıyı maskeler. |
| `SMTP_HOST` | `MAILER=smtp` iken | — | SMTP sunucusu |
| `SMTP_PORT` | hayır | `587` | SMTP kapısı |
| `SMTP_USER` | hayır | — | SMTP kullanıcı adı |
| `SMTP_PASSWORD` | hayır | — | SMTP parolası. Günlüğe yazılmaz. |
| `SMTP_FROM` | `MAILER=smtp` iken | — | Gönderen adresi |
| `ARGON2_TIME` | hayır | `2` | argon2id yineleme sayısı |
| `ARGON2_MEMORY` | hayır | `19456` | argon2id bellek (KiB) |
| `ARGON2_THREADS` | hayır | `1` | argon2id paralellik |
| `JWT_SECRET` | evet | — | HS256 imza anahtarı. Eksikse süreç açılmaz. Günlüğe yazılmaz. |
| `LLM_ENDPOINT_URL` | hayır | — | HuggingFace Inference Endpoint 1 |
| `LLM_TOKEN` | hayır | — | Uç 1 jetonu. Günlüğe yazılmaz. |
| `LLM_ENDPOINT_URL_2` | hayır | — | İkinci uç. Boşsa yönlendirici tek uçla çalışır. |
| `LLM_TOKEN_2` | hayır | — | Uç 2 jetonu. Günlüğe yazılmaz. |
| `LLM_MAX_TOKENS` | hayır | `600` | `max_new_tokens` |
| `LLM_TIMEOUT_SECONDS` | hayır | `240` | Uç HTTP zaman aşımı |
| `LLM_MAX_CONCURRENCY` | hayır | `4` | Eşzamanlı çıkarım üst sınırı |
| `LLM_DEBUG_DUMP` | hayır | kapalı | Geliştirmede maskelenmiş istek ve model çıktısını `UPLOAD_DIR` altına yazar |
| `UPLOAD_DIR` | hayır | `data/uploads` | PDF ve debug dump dizini |

Yük testi MFA'yı her `girisYap` çağrısında yeniler; bu yüzden giriş ile koşu ayrı adımlardır. `LOADTEST_EPOSTA` ve `LOADTEST_SIFRE` ortamda (veya `.env` içinde) olmalı.

```sh
make loadtest-giris
# stdout'taki geçici jetonu kopyalayın; MFA kodunu API günlüğünden alın
make loadtest TOKEN=<gecici> MFA=<kod> SOZLESME=../testdata/sozlesmeler/argos-megep.pdf ESZAMANLI=5 TEKRAR=2
```

Tarayıcıda açık oturumun erişim jetonunu verirseniz giriş adımı atlanır:

```sh
make loadtest ERISIM=<erisim-jetonu> SOZLESME=../testdata/sozlesmeler/argos-megep.pdf
```

