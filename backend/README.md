# backend

Kontrata Go servisi. GraphQL API (Chi router, gqlgen) ve iş kuralları burada yaşar.

Go modül yolu: `github.com/oz-fatma/kontrata/backend`.
İç paketler `internal/` altında durur.

## Gereksinimler

- Go 1.26 veya üzeri

## Nasıl çalıştırılır

Ortam değişkenleri için `.env.example` dosyasına bakın. Gerçek `.env` dosyasını kendiniz oluşturun; uygulama ortam değişkenlerini doğrudan okur.

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
| `GET` | `/healthz` | Sağlık denetimi. `{"status":"ok","version":"..."}` döner. |

## Ortam değişkenleri

| Değişken | Zorunlu | Varsayılan | Açıklama |
| --- | --- | --- | --- |
| `PORT` | hayır | `8080` | HTTP dinleme kapısı |
| `MONGO_URI` | hayır | — | MongoDB bağlantı adresi. Aşama 2'de kullanılacak. |
