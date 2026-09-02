# ml

Sentetik kontenjan sözleşmesi üretimi, model fine-tune ve değerlendirme.

Eğitim verisi `data/`, ağırlıklar `models/` altındadır; ikisi de sürüme alınmaz.
HuggingFace Inference Endpoint üzerindeki fine-tune edilmiş model buradan beslenir.

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
- `meta.sablon_no` — 1 madde madde, 2 tablo, 3 paragraf, 4 karışık, 5 ekler, 6 mektup.
- `meta.dil` — `tr` veya `en` (hedef dağılım yaklaşık %60 / %40).
- `meta.eklenen_gurultu` — `eksik_alan:...`, `tarih_celiskisi:...`,
  `fiyat_oda_uyusmazligi`, `fazladan_madde`, `bicim_yapisik` (sayı ile para
  birimi bitişik, örn. `161GBP`). Boş liste gürültüsüz örnek demektir.
  İngilizce metindeki oda tipi adları çevrilir (`standart` → `standard`);
  `cikti.oda_tipi` şema değerinde kalır.

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
uyuşmazlığı) kırılır. Argos örneği Aşama 8 denetçi kuralları ve elle değerlendirme
için tutulur.
