# Kontrata masaüstü

Windows ve macOS için Electron kabuğu. Go API alt süreç olarak çalışır; Next.js static export `file://` yerine yerel `kontrata://` şemasıyla yüklenir (mutlak `/giris/` yolları için). Sözleşme verisi tesisten çıkmaz.

## Geliştirme

Üç süreç birden gerekir: MongoDB, Next.js (`localhost:3000`), Electron (API'yi `127.0.0.1:17890` üzerinde kendisi açar).

```sh
# 1. MongoDB (replica set rs0)
docker compose up -d

# 2. Go ikilisi (Electron bunu alt süreç olarak başlatır)
cd backend
make build

# 3. Arayüz
cd web
npm install
npm run dev

# 4. Kabuk — ayrı bir uçbirimde, web ayaktayken
cd desktop
npm install
npm run dev
```

İlk açılışta kurulum ekranı `MONGO_URI`, `LLM_ENDPOINT_URL` ve `LLM_TOKEN` ister. URI kaydedilmeden önce `/healthz` 200 dönene kadar (en fazla 30 sn) yoklanır. Ayarlar ve yenileme jetonu `app.getPath('userData')` altında `safeStorage` ile şifrelenir.

Geliştirmede (`NODE_ENV=development`) Go sürecinin stdout/stderr çıktısı Electron uçbirimine `[backend]` önekiyle düşer. MFA kodu ve doğrulama linki ConsoleMailer ile buraya yazılır. Üretim paketinde bu aktarım kapalıdır.

Tarayıcıdan `http://localhost:3000` ile çalışmaya devam etmek için API'yi ayrıca `make run` (`:8080`) ile açın. Electron içindeki arayüz her zaman gömülü API'ye (`:17890`) gider.

## Paketleme

Kod imzalama yoktur. macOS Gatekeeper ve Windows SmartScreen uyarı verir; bu bilinçli bir sınırlamadır.

```sh
# Arayüz static export
cd web && npm run build

# Platform ikilileri
cd backend
make build-darwin    # bin/api-darwin-arm64 ve api-darwin-amd64
make build-windows   # bin/api-windows-amd64.exe

cd ../desktop
npm install
npm run package:mac   # dmg, arm64 + x64
npm run package:win   # nsis, x64 (macOS/Linux'ta NSIS için Wine gerekebilir)
```

Çıktı `desktop/dist/` altındadır. `beforePack` `web/out` içeriğini `resources/web` altına, ilgili `api-darwin-*` / `api-windows-amd64.exe` dosyasını `resources/bin/api` (veya `api.exe`) olarak kopyalar.

## Betikler

| Betik | Amaç |
| --- | --- |
| `npm run dev` | TypeScript derle, geliştirme Electron'u aç (`localhost:3000`) |
| `npm run build` | Ana süreç ve preload derlemesi |
| `npm run package:mac` | macOS dmg (arm64 ve x64) |
| `npm run package:win` | Windows NSIS (x64) |

## Bilinen sınırlamalar

- **Kod imzalama yok.** Sertifika süreci uzun ve ücretlidir; paketler imzasız dağıtılır.
- **Otomatik güncelleme yok.** Yeni sürüm elle indirilir.
- **MongoDB kullanıcı ortamındadır.** Uygulama kendi veritabanını gömmez. Yerelde `docker compose up -d` (tek düğümlü replica set `rs0`) yeterlidir; üretim tesisinde kendi Mongo'nuz gerekir.
- **LLM uç noktası sizin.** HuggingFace Inference Endpoint adresi ve jetonu kurulum ekranından girilir; boş bırakılırsa yüklenen sözleşmeler `HATA` durumuna geçer.
- Paketlenmiş arayüz `kontrata://app/` ile yüklenir, ham `file://` ile değil. Next.js static export mutlak yolları (`/giris/`) `file://` altında kırılır.
- macOS'ta imzasız uygulamayı açmak için Sistem Ayarları > Gizlilik ve Güvenlik üzerinden izin vermek gerekir.
- Windows paketini macOS'tan üretmek NSIS için Wine isteyebilir; aksi halde Windows makinede `package:win` çalıştırın.
- Ayarları sıfırlamak için kullanıcı veri dizinindeki `settings.bin` silinir (macOS: `~/Library/Application Support/Kontrata`).
