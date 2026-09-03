# Teslim kontrol listesi

Değerlendiren kişi bu tablodan gidebilir. “Dosya / ekran” sütunu kanıtın
nerede olduğunu gösterir.

| Zorunlu madde | Karşılık |
| --- | --- |
| Masaüstü uygulama; sözleşme verisi tesiste kalır | Electron kabuk (`desktop/`). Karar 1, 26. README “Neden masaüstü” |
| Okuyucu: PDF → şema | `backend/internal/agent` Okuyucu, `internal/pdf`, `internal/extract`. Arayüz: yükleme sonrası liste/detay |
| Denetçi: çelişki, eksik, risk | Kural motoru `internal/agent/rules.go`; LLM denetçi ayrı prompt. Detay ekranı `bulgular`. Karar 23 |
| Onarım katmanı (bozuk JSON / şema) | `internal/extract` RepairJSON + Normalize. Test: `extract_test.go` |
| Model: HF Inference Endpoint, fine-tune Qwen | `internal/llm`, `ml/train_colab.ipynb`. README Ölçümler |
| İki uç, yüke göre yönlendirme | `internal/llm/router.go`, `LLM_ENDPOINT_URL_2`. Karar 29 |
| Eşzamanlı çıkarım kuyruğu | `LLM_MAX_CONCURRENCY`, durum **Sırada** (`YUKLENDI`). `extract_job.go` |
| LLM izleme tesiste (Langfuse yok) | `llm_cagrilari`, `llmMetrikleri`. Yönetici → Metrikler. Karar 28 |
| Maskeleme kapatılamaz | `internal/mask`. `docs/ornek-maskelenmis-metin.md`. Karar 27 |
| GraphQL + gqlgen | `backend/graph/schema.graphqls`, `make generate` |
| MongoDB replica set | `docker-compose.yml` `rs0`. Hesap silme transaction. Karar 8–9 |
| argon2id şifre | `internal/auth`, `ARGON2_*`. Kayıt formu |
| MFA (6 hane, 120 sn) | `girisYap` / `mfaDogrula`. Giriş ekranı |
| Jeton rotasyonu | Erişim JWT 15 dk, yenileme 7 gün, tek kullanımlık. Karar 7 |
| Kayıtlı cihaz | `X-Device-Id`, `cihazlarim`. Ayarlar |
| Kiracı yalıtımı | `organizasyonId` süzgeci resolver'da. `TestOrganizationFlow` |
| Yetki resolver'da | SAHIP / YONETICI / GORUNTULEYICI. Arayüz gizlemek yetmez. Karar 10 |
| Denetim kaydı | `denetim_kayitlari`. Kimlik, silme, onay, üye |
| Logda sözleşme/PII yok | ErrorPresenter, maske sayacı. `kvkk.md` Loglama |
| Hesap sayımı koruması | `kayitOl` aynı yanıt; e-posta loglanmaz |
| Hesap silme zinciri | `hesapSil`. Ayarlar → hesabı sil. `TestKVKKSilmeZinciri` |
| Denetim anonimleşir, silinmez | `kullaniciId=silinmis`, IP/UA boş. Karar 8 |
| Veri ihracı | `verilerimiIndir`. Ayarlar. `TestKVKKVeriIhraci` |
| PDF yerel disk, hesap silinince dosya da | `UPLOAD_DIR`, `filestore`. Karar 17 |
| Çalışma zamanı prompt / ayar | Yönetici paneli Promptlar + Ayarlar. Yalnızca SAHIP |
| Next.js static export | `web/` `output: 'export'`. Detay `?id=` |
| Electron Windows + macOS | `desktop/dist/Kontrata-0.1.0-arm64.dmg` (113 MB), `Kontrata-0.1.0.dmg` (118 MB), `Kontrata Setup 0.1.0.exe` (91 MB). Windows kurulumu macOS'ta üretildi, Windows'ta çalıştırılmadı |
| Kod imzalama / auto-update yok | Bilinçli sınırlama. Karar 26 |
| Yük testi komutu | `make loadtest-giris` / `make loadtest`. `docs/yuk-testi-*.md` |
| Mimari kararlar tarihli | `docs/kararlar.md` |
| KVKK belgesi | `docs/kvkk.md` |
| Mimari diyagram | `docs/mimari.md` |
| Kapsam dışı / model notları | `docs/kapsam.md` |

## Aşama 13 kanıtları

| Madde | Kanıt |
| --- | --- |
| Silme zinciri | `backend/graph/kvkk_denetim_test.go` `TestKVKKSilmeZinciri` |
| Veri ihracı | aynı dosya `TestKVKKVeriIhraci` |
| Maske örneği | `docs/ornek-maskelenmis-metin.md` |
| Teslim bu liste | `docs/teslim.md` |
