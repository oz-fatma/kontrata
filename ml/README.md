# ml

Sentetik kontenjan sözleşmesi üretimi, model fine-tune ve değerlendirme.

Eğitim verisi `data/` ve ağırlıklar `models/` sürüme alınmaz. Değerlendirme
çıktıları `results/` altındadır ve **sürüme girer** (gitignore'a eklenmez);
eğitim koşularını ve endpoint metriklerini burada tutarız.

HuggingFace Inference Endpoint üzerindeki fine-tune edilmiş model
(`oz-fatma/kontrata-qwen-merged-v1`) buradan beslenir. Eğitim yerel Mac'te
çalışmaz (8 GB RAM yetersiz); Colab GPU kullanılır.

## Üretim

Şemadan rastgele geçerli bir kontrat nesnesi üretilir, ardından LLM kullanılmadan
şablonla doğal dil metnine çevrilir. Aynı `--seed` aynı jsonl dosyalarını verir.

```sh
cd ml
python3 -m venv .venv
source .venv/bin/activate
pip install -r requirements.txt
python generate.py --seed 42
python validate.py
```

`generate.py` varsayılanı 320 eğitim + 80 doğrulama satırıdır (`ml/data/train.jsonl`,
`ml/data/val.jsonl`). Sayıları `--train` / `--val` ile değiştirilebilir.

## Veri formatı

Her satır bir JSON nesnesidir:

```json
{
  "metin": "KONTENJAN SÖZLEŞMESİ\n...",
  "cikti": { "donem": {}, "oda_kontenjanlari": [], "fiyatlar": [], "release": {}, "stop_sale": [] },
  "meta": { "sablon_no": 1, "dil": "tr", "eklenen_gurultu": [] }
}
```

- `metin` — modele giden sözleşme düz yazısı (Türkçe veya İngilizce şablon).
- `cikti` — `schema/kontrat.json` ile doğrulanmış altın etiket. Tarihler ISO-8601,
  sayılar ham sayı; biçim karışıklığı yalnızca `metin` içindedir.
- `meta.sablon_no` — Türkçe: 1 madde madde, 2 tablo, 3 paragraf, 4 karışık, 5 ekler, 6 mektup. İngilizce: 1 ARTICLE N sözleşmesi, 2 teyit yazısı, 3 schedule ekleri.
- `meta.dil` — `tr` veya `en` (hedef dağılım yaklaşık %50 / %50).
- `meta.eklenen_gurultu` — `eksik_alan:...`, `tarih_celiskisi:...`,
  `fiyat_oda_uyusmazligi`, `fazladan_madde`, `bicim_yapisik` (sayı ile para
  birimi bitişik, örn. `161GBP`). Boş liste gürültüsüz örnek demektir.
  İngilizce metinde oda tipleri İngilizce yazılır (`standard`, `family`,
  `suite`, `junior suite`, `penthouse`, `honeymoon`, `accessible`);
  `cikti.oda_tipi` Türkçe şema değerinde kalır (`standart`, `aile`, `suit`, …).

Kasıtlı gürültü şemayı bozmaz: eksik alan null veya anahtarın düşürülmesiyle
gösterilir (Argos örneğindeki `donem.baslangic: null` gibi); tarih çelişkisi
geçerli `date` dizgeleridir, sıra yanlıştır; fiyat tablosundaki oda tipi
kontenjan listesinde olmayabilir.

## MEGEP örneği neden ayrı durur

`schema/ornek-argos.json`, MEB MEGEP "Paket Tur Üretimi" modülündeki Side Turizm–
Argos Otel örnek sözleşmesinin şemaya işlenmiş halidir. Kamuya açık tek tam
örnektir. Eğitim ve doğrulama jsonl dosyalarına **kopyalanmaz**; `validate.py`
onu yalnızca şema fixture'ı olarak kontrol eder.

Aksi halde model bu tek gerçek metni ezberler, sentetik şablonlara aşırı uyar
ve gerçek operatör kontratında (farklı madde sırası, "normal oda" / "tek kişilik"
uyuşmazlığı) kırılır. Argos örneği Aşama 8 denetçi kuralları ve `evaluate.py`
içindeki ayrı raporda tutulur.

## Colab fine-tune

`train_colab.ipynb` Google Colab T4 (veya üzeri) GPU içindir. Yerelde çalıştırmayın.

1. `python generate.py --seed 42` ile `data/train.jsonl` ve `data/val.jsonl` üretin
   (veya defterde `VERI_KAYNAGI = "github"` bırakıp klon + üretim yaptırın).
2. Colab'da Runtime → Change runtime type → GPU.
3. 🔑 Secrets'a `HF_TOKEN` ekleyin (Hugging Face yazma yetkisi). Jeton deftere
   yapıştırılmaz; `userdata.get("HF_TOKEN")` okur.
4. Defteri açıp hücreleri sırayla çalıştırın:

| Hücre | İş |
|---|---|
| 1 | Paket kurulumu, `nvidia-smi`, HF oturumu |
| 2 | jsonl yükleme (`files.upload`) veya GitHub klonu + `generate.py`; sohbet formatı |
| 3 | `Qwen/Qwen2.5-1.5B-Instruct`, 4-bit nf4 + double quant, LoRA r=16 α=32 |
| 4 | SFTTrainer, 3 epoch, batch 4 × grad acc 4, lr 2e-4 cosine, `max_seq_length` 2048; epoch sonu val kaybı ve süre |
| 5 | val'den 20 örnek: geçerli JSON, şema uyumu, alan doğruluğu |
| 6 | Adapter `oz-fatma/kontrata-qwen-lora-v1`; birleşik `oz-fatma/kontrata-qwen-merged-v1` |

Prompt: sistem talimatı şema özetini (alan adları ve tipler) gömer; kullanıcı
sözleşme metnini, asistan altın JSON'u verir.

Hub'a basılan birleşik repo Inference Endpoint'e bağlanır. Adapter ayrı durur
ki LoRA yeniden eğitilebilsin.

## Değerlendirme (yerel)

Endpoint ayağa kalktıktan sonra Mac'te yalnızca HTTP istemcisi çalışır; model
yerelde yüklenmez.

```sh
cd ml
python evaluate.py \
  --endpoint https://YOUR-ENDPOINT.huggingface.cloud \
  --token "$HF_TOKEN" \
  --data data/val.jsonl \
  --concurrency 4
```

`--api chat` (varsayılan) OpenAI uyumlu `/v1/chat/completions` çağırır;
TGI `/generate` için `--api tgi`.

Çıktı: `results/eval_<UTC-tarih>.json` ve aynı metriklerin tablo dökümü.
Dosyada sözleşme gövdesi yoktur; örnek başına yalnızca eşleşme bayrakları vardır.

Argos satırı `schema/ornek-argos.json` altın etiketine karşı ayrı hesaplanır.
Kaynak metin ders kitabı kopyası değildir; altın alandaki tablo ve `kaynak_ifade`
cümlelerinden MEGEP tarzı düz yazı üretilir (eğitim jsonl'ine yine karışmaz).

## Metrikler

Hepsi [0, 1] oranıdır; tabloda yüzde basılır.

| Metrik | Anlamı |
|---|---|
| geçerli JSON | Çıktıdan bir JSON nesnesi ayrıştırılabildi (markdown çiti soyulur). |
| şema uyum | Ayrışan nesne `schema/kontrat.json` (draft 2020-12) doğrulamasından geçti. |
| `donem` | `baslangic` / `bitis` / `alt_donemler` (ad + tarihler) eşit. |
| `oda_kontenjanlari` | `(oda_tipi, adet)` çoklusu eşit; `aciklama` yok sayılır. |
| `fiyatlar` | `oda_tipi`, `tutar`, `birim`, `pansiyon`, `alt_donem_ad` eşit. |
| `release` | `gun` ve `kapsam` eşit; `kaynak_ifade` yok sayılır. |
| `stop_sale` | tarih aralığı + kapsam + bildirim yöntemi; sıra önemsiz. |

Şema uyumu gerçeği garantilemez: uydurulmuş ama şema-geçerli bir tarih yine
hatalıdır. Alan doğruluğu altın etiketle kıyaslar. Argos satırı sentetik val
ortalamasından bağımsızdır; ezber yoksa burada düşüş beklenir.

Örnek raporlar için `results/` dizinine bakın.
