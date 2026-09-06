# Kontrata — Jüri kurulum rehberi

Bu belge değerlendiren kişinin uygulamayı ayağa kaldırması içindir.
Gizli değerler (`LLM_TOKEN`, canlı endpoint) bu dosyada **yoktur**;
ayrı özel kanalda iletilir veya jüri kendi HuggingFace ucunu bağlar.

| Kaynak | Link |
| --- | --- |
| Depo | https://github.com/oz-fatma/kontrata |
| Sürüm paketleri | https://github.com/oz-fatma/kontrata/releases/tag/v0.1.0 |
| Mimari | [`docs/mimari.md`](mimari.md) |
| Teslim listesi | [`docs/teslim.md`](teslim.md) |
| KVKK | [`docs/kvkk.md`](kvkk.md) |

---

## 1. Ürün nedir (30 saniye)

Kontrata, otellerin tur operatörlerinden aldığı **kontenjan sözleşmesi PDF**lerini
okuyup yapılandırılmış JSON’a çeviren **masaüstü** uygulamadır.

1. Kullanıcı PDF yükler.
2. **Okuyucu** (fine-tune LLM) şemaya çıkarır; bozuk JSON onarılır.
3. **Denetçi** kural motoru (ve isteğe bağlı LLM) çelişki / eksik / risk arar.
4. Kullanıcı alanları inceler, düzeltir, onaylar.

Sözleşme dosyası ve MongoDB **tesiste** kalır. Modele giden tek çıkış,
zorunlu maskelemeden geçmiş metindir (e-posta / telefon / TCKN örtülür).

---

## 2. Ne indirilir

Release **v0.1.0**:

| Platform | Dosya | Not |
| --- | --- | --- |
| macOS Apple Silicon (M1/M2/M3…) | `Kontrata-0.1.0-arm64.dmg` | Önerilen (arm64 Mac) |
| macOS Intel | `Kontrata-0.1.0.dmg` | x64 |
| Windows x64 | `Kontrata.Setup.0.1.0.exe` | macOS’ta üretildi; Windows’ta imza uyarısı olabilir |

Kod imzalama / otomatik güncelleme **yoktur** (bilinçli sınırlama).

### macOS: “geliştirici doğrulanamadı”

1. Finder’da `Kontrata.app` → **sağ tık** → **Aç** → yine **Aç**, veya
2. **Sistem Ayarları** → **Gizlilik ve Güvenlik** → **Yine de Aç**.

DMG’den uygulamayı **Applications** klasörüne sürüklemeniz önerilir.

---

## 3. Zorunlu dış bağımlılıklar

Uygulama kendi Mongo’sunu veya LLM’ini gömmez.

### 3.1 MongoDB (zorunlu)

- Sürüm: **MongoDB 8**
- **Replica set** gerekir (`rs0`). Hesap silme transaction kullanır.
- Yerel geliştirme / jüri denemesi için depodaki Compose yeterlidir:

```sh
git clone https://github.com/oz-fatma/kontrata.git
cd kontrata
docker compose up -d
```

**Kurulum ekranına yazılacak URI örneği:**

```text
mongodb://localhost:27017/?replicaSet=rs0
```

Compose ayağa kalkınca bir süre `rs.initiate` bekleyebilir; Electron kurulum
ekranı `/healthz` 200 olana kadar (en fazla ~30 sn) yoklar.

Kendi Mongo’nuz varsa aynı şekilde replica set URI verin. URI’yi günlük veya
ekrana yapıştırmayın; yalnızca kurulum alanına girin.

### 3.2 HuggingFace Inference Endpoint (PDF çıkarımı için zorunlu)

- Taban / fine-tune: **Qwen2.5-1.5B-Instruct** → birleşik repo
  `kontrata-qwen-merged-v1` (eğitim: `ml/train_colab.ipynb`).
- Uygulama `LLM_ENDPOINT_URL` + `LLM_TOKEN` ile HTTP çıkarım yapar.
- İkinci uç isteğe bağlı (`LLM_ENDPOINT_URL_2` / `LLM_TOKEN_2`); yük
  dağıtımı için.

**Bu rehberde gerçek URL ve jeton yoktur.**

- Jüri denemesi için değerler **özel mesaj / e-posta** ile iletilir, veya
- Jüri kendi HF endpoint’ini `…-merged-v1` (veya uyumlu chat modeli) ile kurar.

Form / genel kanala **token yazmayın**.

### 3.3 E-posta (SMTP) — isteğe bağlı ama demoda önemli

| Durum | Sonuç |
| --- | --- |
| SMTP boş | `MAILER=console`. Geliştirmede kodlar konsola düşer; **paketlenmiş** uygulamada log çoğu zaman görünmez → doğrulama / MFA zorlaşır. |
| SMTP dolu | Gerçek e-posta ile kayıt doğrulama, şifre sıfırlama, MFA. |

Kurulum ekranı alanları: `SMTP_HOST`, `SMTP_PORT`, `SMTP_USER`,
`SMTP_PASSWORD`, `SMTP_FROM`. `SMTP_HOST` + `SMTP_FROM` doluysa SMTP açılır.

---

## 4. Masaüstü ilk kurulum adımları

1. Mongo’yu başlatın (`docker compose up -d`).
2. DMG/EXE ile Kontrata’yı açın.
3. Kurulum ekranını doldurun:

| Alan | Zorunlu | Örnek / not |
| --- | --- | --- |
| `MONGO_URI` | Evet | `mongodb://localhost:27017/?replicaSet=rs0` |
| `LLM_ENDPOINT_URL` | Evet (çıkarım için) | Özel kanaldaki HF kök URL |
| `LLM_TOKEN` | Evet (çıkarım için) | Özel kanaldaki jeton |
| SMTP_* | Hayır | Demo hesabı için önerilir |

4. Kayıt ol → e-postayı doğrula → giriş → **MFA** (6 hane, ~120 sn).
5. Ana ekrandan PDF yükle (ör. `testdata/sozlesmeler/argos-megep.pdf`).
6. Durum: **Sırada** → **İşleniyor** → **İncelenmeyi bekliyor** (veya HATA).

Önerilen çıkarım modu (geliştirme `.env` / ortam): `EXTRACT_MODE=chunked`
(dört paralel parça: dönem+kontenjan, fiyatlar, release/stop-sale, meta).
Masaüstü kurulum UI’sinde bu alan yoksa backend varsayılanı `single` olabilir;
jüri özel ortamında `chunked` açılması önerilir.

---

## 5. Geliştirme ortamı (tarayıcı + API)

Paket yerine kaynak çalıştırmak için:

```sh
git clone https://github.com/oz-fatma/kontrata.git
cd kontrata
docker compose up -d

cd backend
cp .env.example .env
# JWT_SECRET (en az 32 karakter), MONGO_URI
# LLM_ENDPOINT_URL, LLM_TOKEN
# EXTRACT_MODE=chunked   # önerilir
# MAILER=console         # MFA kodu backend günlüğünde
make run                 # http://localhost:8080

cd ../web
npm install
npm run dev              # http://localhost:3000
```

Sağlık: `GET http://localhost:8080/healthz` → `database: connected`.

`MAILER=console` iken doğrulama linki ve MFA kodu **backend terminal**inde
görünür (alıcı maskeli).

Electron geliştirme: `desktop/README.md` (API `:17890`, arayüz `localhost:3000`).

---

## 6. Ortam değişkenleri özeti

| Değişken | Zorunlu | Açıklama |
| --- | --- | --- |
| `MONGO_URI` | Evet | Replica set URI |
| `MONGO_DATABASE` | Hayır | Varsayılan `kontrata` |
| `JWT_SECRET` | Dev `.env` | HS256; eksikse süreç açılmaz |
| `LLM_ENDPOINT_URL` / `LLM_TOKEN` | Çıkarım için | HF uç 1 |
| `LLM_ENDPOINT_URL_2` / `LLM_TOKEN_2` | Hayır | HF uç 2 |
| `LLM_MAX_CONCURRENCY` | Hayır | Varsayılan 4 |
| `EXTRACT_MODE` | Hayır | `single` (varsayılan) veya `chunked` |
| `MAILER` | Hayır | `console` / `smtp` |
| `APP_URL` | Hayır | Doğrulama link kökü (`http://localhost:3000`) |
| `UPLOAD_DIR` | Hayır | PDF dizini |

Ayrıntı: `backend/.env.example`, kök `README.md`.

---

## 7. Test hesabı

Paylaşılan kalıcı demo kullanıcısı **yoktur**.

Jüri:

1. Kendi e-postasıyla kayıt olur (bireysel veya kurumsal).
2. Doğrulama + MFA tamamlar.
3. Örnek PDF yükler.

Kurumsal hesapta Üyeler / Yönetici (prompt, metrik) menüleri açılır.
Bireyselde sözleşme listesi, detay ve Ayarlar → Genel yeterlidir.

---

## 8. Örnek sözleşmeler (repoda)

| Dosya | Not |
| --- | --- |
| `testdata/sozlesmeler/argos-megep.pdf` | Basit / eğitim dışı MEGEP örneği |
| `testdata/sozlesmeler/` altı diğer PDF’ler | Daha zor operatör örnekleri |

Model 1.5B sınırında olabilir; karmaşık tablolarda alan eksikleri görülebilir.
`EXTRACT_MODE=chunked` bunu kısmen iyileştirir. Kullanıcı onayı her zaman son adımdır.

---

## 9. Bilinen sınırlamalar (dürüstçe)

- Kod imzası yok → Gatekeeper / SmartScreen uyarısı.
- Windows paketi Windows makinede resmi test edilmedi.
- Taranmış (OCR’siz) PDF desteklenmez.
- LLM denetçi katmanı pratikte az bulgu üretebilir; asıl ağ **kural motoru**.
- `LLM_TOKEN` ve canlı endpoint bu belgede yoktur.

---

## 10. Jüriye özel kanalda iletilecekler (şablon)

Aşağıyı **Form’a değil**, özel mesaja yapıştırın; değerleri siz doldurun:

```text
Kontrata jüri erişimi
- MONGO (yerel): mongodb://localhost:27017/?replicaSet=rs0
  (önce: docker compose up -d)
- LLM_ENDPOINT_URL: <HF_ENDPOINT_KOK_URL>
- LLM_TOKEN: <HF_TOKEN>
- Önerilen: EXTRACT_MODE=chunked
- Release: https://github.com/oz-fatma/kontrata/releases/tag/v0.1.0
```

---

## 11. Hızlı kontrol listesi

- [ ] Docker Mongo `rs0` ayakta
- [ ] DMG/EXE kuruldu / Gatekeeper izni verildi
- [ ] Kurulumda MONGO + LLM URL + TOKEN
- [ ] Kayıt + doğrulama + MFA
- [ ] Örnek PDF yüklendi, durum incelenmeyi bekliyor
- [ ] (İsteğe bağlı) SMTP veya console ile e-posta akışı doğrulandı
