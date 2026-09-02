# Kontrata

[![CI](https://github.com/oz-fatma/kontrata/actions/workflows/ci.yml/badge.svg)](https://github.com/oz-fatma/kontrata/actions/workflows/ci.yml)

Otellerin tur operatörlerinden aldığı kontenjan sözleşmelerini okuyup yapılandırılmış veriye çeviren masaüstü uygulaması.

## Dizin yapısı

| Dizin | Amaç |
| --- | --- |
| `backend/` | Go servisi |
| `web/` | Next.js arayüz |
| `desktop/` | Electron kabuk |
| `ml/` | sentetik veri üretimi, fine-tune, değerlendirme |
| `docs/` | mimari kararlar ve KVKK notları |

## Gereksinimler

## Kurulum

MongoDB'yi yerel geliştirme için Docker ile başlatın:

```sh
docker compose up -d
```

Veritabanı `localhost:27017` üzerinde, `kontrata` adlı veritabanı ile ayağa kalkar. Kimlik doğrulama bu ortamda kapalıdır.

Hesap silme gibi çok belgelik işlemler MongoDB transaction kullanır; bu yüzden konteyner tek düğümlü replica set (`rs0`) olarak çalışır. Sağlık kontrolü `rs.initiate()` sonrası birincil olana kadar bekler. Daha önce replica setsiz ayağa kalkmış bir volume varsa konteyneri `--replSet rs0` ile yeniden oluşturun (`docker compose up -d --force-recreate`). Veri dosyası uyumsuzsa volume'u silip baştan açın (`docker compose down -v`).

## Geliştirme

```sh
docker compose up -d
cd backend
export MONGO_URI=mongodb://localhost:27017
make run
```

Sağlık denetimi: `GET http://localhost:8080/healthz`. Mongo erişilemezse süreç yine de dinler; uç `503` ve `"status":"degraded"` döner.

## Mimari

## Bilinen sınırlamalar
