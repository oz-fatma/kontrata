# Kararlar

## 1. Masaüstü uygulaması tercihi
Tarih: 2026-09-01
Durum: kabul edildi
Bağlam: Otel-tur operatörü sözleşmelerinde fiyat ve kontenjan bilgisi ticari sırdır; oteller bu veriyi bulut hizmetine yüklemek istemez. Rakip ürünler bulut SaaS olduğu için bu segmente giremiyor.
Karar: Electron ile masaüstü uygulaması, veri tesiste kalır.
Sonuç: Dağıtım ve güncelleme daha zor, kod imzalama gerekir; buna karşılık gizlilik doğrudan satış argümanı olur.

## 2. MongoDB tercihi
Tarih: 2026-09-01
Durum: kabul edildi
Bağlam: Her tur operatörünün sözleşme formatı farklı, çıkarılan alan kümesi sabit değil.
Karar: Doküman veritabanı.
Sonuç: Şema esnekliği kazanıldı, ilişkisel bütünlük garantileri uygulama katmanına taşındı.

## 3. GraphQL tercihi
Tarih: 2026-09-01
Durum: kabul edildi
Bağlam: Sözleşme detay ekranı değişken alan kümeleri çekiyor, liste ekranı ise az alan.
Karar: gqlgen ile şemadan kod üretimi.
Sonuç: Aşırı veri çekimi önlendi; buna karşılık REST'e göre önbellekleme ve hata yönetimi daha karmaşık.

## 4. Sağlık kontrolünde dayanıklılık
Tarih: 2026-09-01
Durum: kabul edildi
Bağlam: MongoDB erişilemez olduğunda servisin davranışı tanımlanmalıydı.
Karar: Açılışta çökme yok; `/healthz` degraded durumu ve HTTP 503 döner, uygulama ayakta kalır.
Sonuç: Kısmi hizmet mümkün, izleme sistemleri durumu ayırt edebilir.

## 5. Go hedef sürümü 1.25
Tarih: 2026-09-02
Durum: kabul edildi
Bağlam: Yerelde Go 1.26 kurulu, ancak golangci-lint henüz 1.26 ile derlenmiş sürüm yayınlamadı; CI lint işi bu yüzden başarısız oluyor.
Karar: `go.mod` hedefi 1.25'e sabitlendi.
Sonuç: Go 1.26 özellikleri kullanılamaz; araç zinciri yakaladığında yükseltilecek.

## 6. Kayıt ve e-posta doğrulama
Tarih: 2026-09-02
Durum: kabul edildi
Bağlam: Kullanıcı hesabı e-posta ile doğrulanacak; hesap varlığı yanıt veya günlüklerden sızmamalı, şifreler geri döndürülemez saklanmalı.
Karar: Şifre argon2id (OWASP varsayılan maliyet) ile PHC biçiminde tutulur. Doğrulama kodu 32 bayt rastgele değer, veritabanında SHA-256. `kayitOl` ve `dogrulamaTekrarGonder` e-posta kayıtlı olsa da aynı yanıtı verir. Denetim kaydına kimlik işlemi yazılır; şifre, token ve e-posta loglanmaz.
Sonuç: GraphQL yüzeyi kullanıcı veya oturum token'ı dönmez; geliştirmede ConsoleMailer alıcıyı maskeleyerek iletiyi basar.

## 7. MFA ve oturum jetonları
Tarih: 2026-09-02
Durum: kabul edildi
Bağlam: Masaüstü istemcinin API'ye kimlik doğrulaması gerekir; şifre tek başına oturum açmamalı.
Karar: Girişte 6 haneli MFA (120 sn, 5 deneme). Erişim jetonu JWT HS256 15 dk; yenileme jetonu 32 bayt rastgele, 7 gün, rotasyonlu. JWT'de e-posta yok. Sözleşme alanları, `cikisYap` ve `oturumlarim` `@auth` ister. `jetonYenile` erişim jetonu istemez; kimlik yenileme jetonunun kendisiyle doğrulanır. `girisYap`, `mfaDogrula`, `kayitOl` ve şifre sıfırlama alanları da `@auth` dışındadır.
Sonuç: `JWT_SECRET` zorunlu; şifre sıfırlama tüm oturumları iptal eder. Süresi dolmuş erişim jetonu yenilemeyi engellemez.

## 8. Hesap silmede denetim kaydı
Tarih: 2026-09-02
Durum: kabul edildi
Bağlam: KVKK md. 7 silme hakkını tanır; md. 12 veri sorumlusuna işleme faaliyetlerini kayıt altında tutma yükümlülüğü getirir. Silme talebinin kendisi de kanıtlanabilir olmalıdır.
Karar: Hesap silinince sözleşmeler, token'lar, MFA kodları, oturumlar, cihazlar ve kullanıcı belgesi tek MongoDB işleminde (transaction) kaldırılır; bu yüzden sunucu replica set (veya mongos) olmalıdır. Yerel Docker ve CI bu yüzden tek düğümlü replica set (`rs0`) olarak çalışır. `denetim_kayitlari` silinmez; `kullaniciId` değeri `silinmis` olur, `ipAdresi` ve `kullaniciAjani` boşaltılır. İşlem `HESAP_SILINDI` olayıyla kapanır.
Sonuç: Kısmi silme olmaz. Denetim izi kişisel veriden arındırılmış biçimde kalır. Replica setsiz MongoDB'de silme işlemi reddedilir.

## 9. CI MongoDB replica set
Tarih: 2026-09-02
Durum: kabul edildi
Bağlam: Hesap silme transaction replica set ister. GitHub Actions `services` bloğu konteynere `mongod --replSet` argümanı geçirmez; `mongo:8` servisi tek düğüm kalır. `mongodb/mongodb-atlas-local` replica set hazır gelir ve servis olarak kullanılabilir, ancak Atlas Search süreci taşır ve yerel `docker-compose` imajından (`mongo:8`) sapar.
Karar: CI, yerel ortamla aynı `mongo:8 --replSet rs0` sürecini iş adımında `docker run` ile başlatır. Testler `rs.status()` ve birincil olana kadar beklemeden başlamaz.
Sonuç: Yerel ve CI aynı motoru ve replica set adını (`rs0`) kullanır. GHA `services` bloğunda Mongo yoktur.

## 10. Organizasyon kapsamı ve roller
Tarih: 2026-09-02
Durum: kabul edildi
Bağlam: Bireysel hesap kendi sözleşmesinin sahibidir; kurumsal hesapta sözleşme kuruma aittir. Çok kiracılı sistemlerde en sık hata başka kiracının belgesini kimlik ile okumaktır.
Karar: `sozlesmeler` koleksiyonuna `organizasyonId` yazılır. Kurumsal listede süzgeç yalnızca bu alandır; bireyselde `kullaniciId` ve boş `organizasyonId`. Resolver, kaydı getirmeden önce kullanıcının organizasyonuyla karşılaştırır; eşleşmezse `null` döner (yokmuş gibi). Rol yetkisi (`SAHIP` / `YONETICI` / `GORUNTULEYICI`) arayüz gizlemesine bırakılmaz, resolver'da kesilir. Mevcut belgeler açılışta kurumsal kullanıcının `organizasyonId` değeriyle doldurulur.
Sonuç: Üye daveti 7 günlük TTL kodla gelir; sahip başka üye varken hesabını silemez, önce devir veya `organizasyonSil` gerekir.

## 11. Sentetik sözleşme verisi şablonla üretilir
Tarih: 2026-09-02
Durum: kabul edildi
Bağlam: Okuyucu agent'ı eğitmek için yüzlerce kontenjan sözleşmesi gerekir; gerçek otel kontratları tesiste kalır ve eğitim kümesine alınamaz. Tek kamuya açık örnek MEGEP Argos sözleşmesidir.
Karar: `ml/generate.py` şemadan rastgele geçerli nesne üretir, metni LLM'siz şablonla yazar. MEGEP örneği (`ornek-argos.json`) eğitim/doğrulama jsonl dosyalarına karışmaz; tutulur ki model tek gerçek örneği ezberlemesin. Kasıtlı gürültü (eksik alan, tarih çelişkisi, fiyat–kontenjan uyuşmazlığı) JSON Schema'yı bozmaz.
Sonuç: Üretim deterministiktir (`--seed`). Üretilen `ml/data/*.jsonl` sürüme girmez.

## 12. Go tanımlayıcıları İngilizce
Tarih: 2026-09-02
Durum: kabul edildi
Bağlam: Go iç kodu ile GraphQL/veritabanı katmanı farklı diller kullanıyordu, tutarsızdı.
Karar: Go tanımlayıcıları İngilizce; GraphQL şeması, MongoDB alanları, yorumlar ve kullanıcıya dönen metinler Türkçe.
Sonuç: Kod uluslararası okunabilir, domain dili sektör terimleriyle örtüşmeye devam ediyor.

## 13. Fine-tune Colab'da LoRA, dağıtım birleşik HF repo
Tarih: 2026-09-02
Durum: kabul edildi
Bağlam: Okuyucu modeli HuggingFace Inference Endpoint'te çalışacak. Yerel Mac 8 GB RAM ile Qwen fine-tune edilemez.
Karar: Eğitim Colab GPU'da `Qwen/Qwen2.5-1.5B-Instruct` + 4-bit LoRA (`ml/train_colab.ipynb`). Adapter `oz-fatma/kontrata-qwen-lora-v1`, birleşik ağırlık `oz-fatma/kontrata-qwen-merged-v1`; endpoint birleşik sürümü kullanır. HF jetonu Colab secrets'tan okunur, koda gömülmez. Yerel `evaluate.py` endpoint'e karşı val.jsonl ve MEGEP Argos örneğini ayrı raporlar. `ml/results/` sürüme alınır; `ml/data/` ve `ml/models/` alınmaz.
Sonuç: Eğitim bulutta, çıkarım endpoint'te, metrikler depoda izlenir.

## 14. PDF metin çıkarma OCR'siz, saf Go
Tarih: 2026-09-02
Durum: kabul edildi
Bağlam: Okuyucu agent PDF sözleşmeyi şemaya çevirecek; Aşama 7'de kaynak sayfa numarası gerekir. Masaüstü dağıtımında poppler/tesseract gibi native bağımlılık istenmez.
Karar: `internal/pdf` `github.com/ledongthuc/pdf` ile sayfa sayfa düz metin çıkarır. Taranmış (metin katmanı olmayan) PDF `ErrNoTextLayer` döner; OCR yok. Sözleşme gövdesi loglanmaz, yalnızca sayfa ve karakter sayısı yazılır. Tekrarlayan başlık/altbilgi ayıklanır; tablo boşlukları ve madde numaraları korunur.
Sonuç: Model çağrısından bağımsız bir çıkarım katmanı var; taranmış evrak ayrı ürün kararı olarak kalır.

## 15. Model çıktısı onarım ve şema doğrulama Go'da
Tarih: 2026-09-02
Durum: kabul edildi
Bağlam: Okuyucu LLM çıktısı markdown, yarım JSON veya şema dışı alan üretebilir. Model henüz bağlanmaz; onarım katmanı bağımsız durmalı.
Karar: `internal/extract` ham metni `RepairJSON` ile nesneye çevirir, `Normalize` şemaya çeker (enum, ISO tarih, sayı, `stop_sale` taşıma), `Validate` gömülü `kontrat.json` ile jsonschema doğrular. Gömme `ml/schema/kontrat.json` kopyasıdır; test kaynakla eşitliği kontrol eder. Sözleşme değeri loglanmaz.
Sonuç: Aşama 7 model çağrısından önce çıktı sözleşmeye hazır bir Go katmanı var.

## 16. Arayüz oturum jetonu
Tarih: 2026-09-02
Durum: kabul edildi
Bağlam: Next.js arayüzü static export ile Electron'a gömülür. Aşama 9'da yenileme jetonu `sessionStorage`'daydı; Aşama 10 masaüstü kabuğunu ekledi.
Karar: Erişim jetonu yalnızca bellek (modül değişkeni) tutulur. Masaüstünde yenileme jetonu preload üzerinden `safeStorage` ile şifrelenip `userData/refresh.bin` dosyasına yazılır; renderer Node API'sine erişmez. Tarayıcıda geliştirmede `sessionStorage` anahtarı `kontrata.refresh` kalır. 401 veya GraphQL `kimlik doğrulaması gerekli` yanıtında `jetonYenile` denenir; başarısızsa girişe yönlendirilir. Cihaz kimliği `localStorage` (`kontrata.device`) ile `X-Device-Id` başlığına yazılır. Tarayıcıdan API'ye istek için CORS, `Origin` yansıtır.
Sonuç: Electron'da oturum uygulama kapanıp açılınca da durur. Tarayıcı sekmesi kapanınca jeton düşer. CORS masaüstü paketinde `kontrata://` kökeni için de yansıtılır.

## 17. PDF yerel diskte, çıkarım asenkron
Tarih: 2026-09-03
Durum: kabul edildi
Bağlam: Okuyucu HuggingFace Inference Endpoint'e bağlanır; soğuk başlangıç 20–60 sn sürebilir. Sözleşme dosyası tesisten çıkmamalı, bulut depolama istenmez.
Karar: `sozlesmeYukle` kaydı `YUKLENDI` ile hemen döner; çıkarım süreç içi kuyrukta `ISLENIYOR` → `INCELENMEYI_BEKLIYOR` veya `HATA`. PDF'ler `UPLOAD_DIR` altında UUID adıyla tutulur, orijinal ad veritabanındadır. Hesap ve organizasyon silinince dosyalar da silinir. Endpoint kök yola `inputs`/`generated_text` ile konuşur; Qwen sohbet şablonu zorunludur. Prompt ve çıktı loglanmaz.
Sonuç: Kullanıcı yüklemede beklemez. Model ve dosya tesiste/yerel endpoint'te kalır.

## 18. CPU Inference Endpoint
Tarih: 2026-09-03
Durum: kabul edildi
Bağlam: GPU endpoint maliyeti yüksek; Okuyucu çıkarımı kullanıcıyı bekletmez (karar 17). CPU üzerinde üretim 60–180 sn, soğuk başlangıçla birlikte 240 sn'ye yaklaşabilir. 1536 token limiti CPU'da süre aşımına yol açıyordu.
Karar: HuggingFace Inference Endpoint CPU olarak tutulur. `max_new_tokens` varsayılan 768 (`LLM_MAX_TOKENS`), HTTP zaman aşımı 240 sn (`LLM_TIMEOUT_SECONDS`). Zaman aşımı yeniden denenmez; yalnızca 503 soğuk başlangıçta 8 deneme yapılır.
Sonuç: Çıkarım 60–180 sn sürebilir. Bu yüzden iş asenkron kalır; kullanıcı yüklemede beklemez.

## 19. Ham model çıktısı yalnızca yerel hata ayıklamada
Tarih: 2026-09-03
Durum: kabul edildi
Bağlam: İlk çıkarım turunda model 830 karakter üretip JSON nesnesi döndürmeyebiliyor; logda yalnızca `neden=nesne_yok` ve `hata=N` görünüyordu. Üretimde model çıktısı ve sözleşme metni loglanmaz (karar 17).
Karar: `LLM_DEBUG_DUMP=true` iken `UPLOAD_DIR` altına `cikarma-{sozlesmeId}-{zaman}-{n}.txt` yazılır. Dosyada modele giden maskelenmiş kullanıcı metni (`=== GONDERILEN (maskelenmis) ===`) ve ham model çıktısı (`=== ALINAN ===`) vardır. `nesne_yok` durumunda ilk 200 karakter loglanır. Varsayılan kapalıdır; üretimde açılmaz. Şema doğrulama hataları alan yolu ve kısıt adıyla loglanır, değerler yazılmaz.
Sonuç: Yerelde hem maskeleme hem model çıktısı dosyadan doğrulanır; üretim günlüklerinde sözleşme içeriği yoktur.

## 20. Çıkarım prompt'u eğitim metninden ayrıldı
Tarih: 2026-09-03
Durum: kabul edildi
Bağlam: Eğitimdeki uzun SYSTEM_PROMPT (şema alan özeti) gerçek PDF metinlerinde markdown tablo çıktısı tetikliyordu; RepairJSON nesne bulamıyordu. Örnek JSON içeren kısa prompt modelin JSON üretmesini sağladı. Model bazen kök nesneyi erken kapatıp fazla `}` bırakıyor; ardışık-nesne birleştirme bunu çözemiyor. 768 token limiti çıktıya gerekenden uzundu.
Karar: Okuyucu sistem prompt'u eğitimdeki metinden bağımsızdır; örnek çıktı ve kısa alan kuralları vardır. RepairJSON önce ilk `{` ile son `}` arasını dener, dengesiz kapanış parantezlerini atar, olmazsa mevcut ardışık-nesne birleştirmeye düşer. `max_new_tokens` varsayılanı 600 (`LLM_MAX_TOKENS`).
Sonuç: Gerçek PDF'te JSON üretimi düzelir; erken kapanmış nesneler onarılır; çıkarım süresi kısalır.

## 21. Static export için sorgu parametreli detay
Tarih: 2026-09-03
Durum: kabul edildi
Bağlam: Next.js `output: 'export'` Electron paketlemesi için zorunlu. Dinamik rota `/sozlesme/[id]` derleme zamanında bilinmeyen kimlikler kullanıyor; `generateStaticParams` placeholder'ı gerçek id ile açılınca sayfa çöküyordu.
Karar: Sözleşme detayı `/sozlesme/?id=` sorgu parametresiyle açılır. `/dogrula` ve `/sifre-sifirla` token'ı zaten sorgu ile alıyordu; yeni dinamik segment eklenmez.
Sonuç: `next build` statik HTML üretir. Derleme zamanında sözleşme listesi gerekmez.

## 22. Eğitim ve üretim prompt'u hizalandı
Tarih: 2026-09-03
Durum: kabul edildi
Bağlam: Karar 20 üretim prompt'unu eğitimden ayırmıştı. İngilizce sözleşmelerde çıkarım zayıf kalıyordu; sentetik küme %60 Türkçe idi ve İngilizce şablonlar TR çevirisi gibi duruyordu.
Karar: `SYSTEM_PROMPT` eğitim defteri, `evaluate.py` ve Okuyucu agent'ta aynıdır. `generate.py` dil oranı %50/%50; İngilizce metinde oda tipleri İngilizce, `cikti` Türkçe kalır.
Sonuç: Yeniden eğitim bu hizalamayı modele taşır.

## 23. Denetçi karma kural + LLM
Tarih: 2026-09-03
Durum: kabul edildi
Bağlam: Okuyucu şemaya çevirir; çelişki, eksik madde ve belirsiz ifade ayrı bir denetim ister. Tamamen LLM hem yavaş hem de tarih/tablo hatalarında tutarsız kalır.
Karar: Denetçi iki katmanlıdır. Altı kural Go'da deterministik çalışır (tarih çelişkisi, alt dönem taşması, fiyat–kontenjan uyuşmazlığı, boş stop-sale, makul olmayan release, zorunlu alan). Yoruma dayalı denetim ayrı sistem prompt'u ve 300 token ile LLM'dedir; kurallarla çakışan konular istenmez. LLM çıktısı ayrışmazsa veya model hata verirse kural bulguları yine yazılır; sözleşme HATA olmaz. Onay kullanıcıdadır: çıkarım başarılıysa durum her zaman INCELENMEYI_BEKLIYOR.
Sonuç: Bulgular sözleşme kaydına ve GraphQL `bulgular` alanına yazılır.

## 24. Sözleşme eylemleri ve kaynak PDF
Tarih: 2026-09-03
Durum: kabul edildi
Bağlam: Detay ekranındaki Onayla, alan düzeltme, JSON indir ve Kaynağı aç düğmeleri bağlı değildi. PDF'i GraphQL üzerinden dönmek hem şemayı şişirir hem de tarayıcıda yeni sekmede açmayı zorlaştırır.
Karar: Onay `sozlesmeOnayla` ile yalnızca `INCELENMEYI_BEKLIYOR` → `ONAYLANDI` (SAHIP/YONETICI). Alan düzeltme `sozlesmeAlanGuncelle` ile güveni 1.0 yapar, `elleDuzeltildi` işaretler, değeri denetim kaydına yazmaz ve Denetçi'yi yeniden çalıştırır. Kaynak PDF `GET /dosya/{id}` REST ucundan Authorization ile servis edilir; başka kiracının kaydı 404 döner.
Sonuç: Onaylı kayıt salt okunur. Liste 3 sn aralıkla koşulsuz yenilenir; durdurma optimizasyonu sonra eklenebilir.

## 25. Yükleme dizini sürüme girmez
Tarih: 2026-09-03
Durum: kabul edildi
Bağlam: `backend/uploads/` bir commit'te yanlışlıkla versiyonlandı, sonraki commit'te takip dışına alındı. Sözleşme dosyası tesiste kalır; sürüme girmemelidir.
Karar: Dizin `.gitignore` içindedir. Geçmişte kalan dosyalar test verisidir (sentetik sözleşmeler), gerçek müşteri verisi değildir. Git geçmişi bu yüzden yeniden yazılmaz.
Sonuç: Yeni yüklemeler commit'e düşmez. Geçmiş blob'lar sentetik/test çıktısıdır.

## 26. Electron kabuk, imzasız paket, gömülü API
Tarih: 2026-09-03
Durum: kabul edildi
Bağlam: Sözleşme verisi tesisten çıkmaz; arayüz masaüstü kabuğunda, Go API aynı makinede alt süreçtir. Next.js static export mutlak yolları (`/giris/`) ham `file://` altında kırılır. Kod imzalama sertifikası uzun ve ücretlidir. Otomatik güncelleme ayrı bir dağıtım altyapısı ister.
Karar: Paketlenmiş arayüz `kontrata://app/` özel şemasıyla `resources/web` dizininden okunur (ağ yok). Geliştirmede `NODE_ENV=development` iken `http://localhost:3000` yüklenir. API `backend/bin` ikilisinden spawn edilir, `/healthz` 200 olana kadar (30 sn) beklenir, kapanışta SIGTERM ve 5 sn sonra SIGKILL. Kullanıcı verisi `app.getPath('userData')`; MongoDB, LLM uç noktası ve jeton ilk açılışta istenir, `safeStorage` ile saklanır. `JWT_SECRET` burada üretilir. `contextIsolation` açık, `nodeIntegration` kapalı. Paketler imzasızdır (`mac.identity` boş); otomatik güncelleme yoktur. Platform ikilileri `make build-darwin` / `make build-windows` ile üretilir.
Sonuç: Windows NSIS (x64) ve macOS dmg (arm64 + x64) üretilebilir. Gatekeeper/SmartScreen uyarısı beklenir. Mongo ve model uç noktası kullanıcı ortamındadır.

## 27. Çalışma zamanı prompt sürümü ve zorunlu maskeleme
Tarih: 2026-09-03
Durum: kabul edildi
Bağlam: Yönetici Okuyucu/Denetçi davranışını kod dağıtmadan değiştirmek ister. Serbest prompt, kişisel verinin modele gidişini zayıflatır.
Karar: Prompt sürümleri `prompt_surumleri` koleksiyonunda organizasyon ve tip (OKUYUCU/DENETCI) başına tutulur; yalnızca bir aktif sürüm vardır, eskiler silinmez. Ayarlar (`denetciRiskEsigi`, `maxToken`) `ayarlar` belgesindedir. GraphQL uçları `@auth` ve yalnızca SAHIP. Organizasyon sürümü yoksa koddaki varsayılan kullanılır; kullanılan Okuyucu sürüm numarası sözleşmeye `promptSurumu` yazılır. `ayarlar` belgesi yoksa ilk okumada varsayılan değerlerle oluşturulur (`guncellemeTarihi` Time! boş kalmaz). Sözleşme metni LLM'e gitmeden önce `internal/mask` e-posta, telefon ve TCKN benzeri desenleri örter; bu katman kapatılamaz. Denetim kaydı tip/sürüm/değişen ayar adını tutar, prompt metnini tutmaz. GraphQL ErrorPresenter bilinmeyen hataları üretimde «işlem tamamlanamadı» yapar; `GRAPHQL_PLAYGROUND=true` iken gqlgen metni korunur. Arayüz geliştirmede (`NODE_ENV=development`) sunucu mesajını gösterir.
Sonuç: Tesis sahibi çıkarımı ayarlayabilir; kişisel veri maskelemesi yöneticiye bağlı değildir. Boş yönetici paneli hata değildir.

## 28. LLMOps izlemesi tesiste kalır
Tarih: 2026-09-03
Durum: kabul edildi
Bağlam: Çıkarım gecikmesi, uç sağlığı ve hata türü ölçülmeli. Langfuse ve benzeri SaaS izleme, sözleşme işinin tesisten çıkmaması ilkesine aykırı ek altyapı ve bağımlılık getirir. İstenen metrik kümesi sınırlıdır (süre, başarı, uç, agent, hata tipi).
Karar: `llm_cagrilari` koleksiyonu kendi kodumuzla tutulur. Prompt ve model çıktısı yazılmaz. Kayıt 90 gün TTL ile silinir. İzleme yazımı başarısız olsa da çıkarım devam eder. GraphQL `llmMetrikleri` / `llmCagrilari` yalnızca SAHIP.
Sonuç: Ölçüm tesiste kalır. Üçüncü parti LLMOps yok.

## 29. Yönlendirme ölçütü aktif istek sayısı
Tarih: 2026-09-03
Durum: kabul edildi
Bağlam: İki HuggingFace CPU ucu farklı ısınma ve gecikme gösterir. Yalnızca ortalama gecikme soğuk başlangıcı cezalandırır; round-robin uçların anlık yükünü yok sayar.
Karar: `internal/llm/router.go` önce sağlıksız uçları eledikten sonra `aktifIstek` en az olanı seçer; eşitlikte son 10 çağrının ortalaması düşük olan kazanır. Üç ardışık soğuk-olmayan hata 60 sn sağlıksızlık işaretler. 503 soğuk başlangıç sayılmaz.
Sonuç: Yük, o an meşgul olmayan uca kayar; ısınmakta olan uç yanlışlıkla cezalandırılmaz.

## 25. Electron'da e-posta doğrulama kodu yapıştırılır
Tarih: 2026-09-03
Durum: kabul edildi
Bağlam: Kayıt sonrası ekranda yalnızca «Girişe git» vardı. Tarayıcıda e-postadaki `/dogrula?token=` bağlantısı çalışır; Electron'da adres çubuğu olmadığı için açılamaz.
Karar: Kayıt sonrası ekran ve `/dogrula` (sorgu token'ı yoksa veya geçersizse) ham kod ya da `?token=` içeren metin yapıştırma alanı sunar. `epostaDogrula` false dönerse başarı gösterilmez. `kontrata://` derin bağlantı bu aşamada yok.
Sonuç: Masaüstünde doğrulama, tarayıcı bağlantısına bağlı olmadan tamamlanır.








