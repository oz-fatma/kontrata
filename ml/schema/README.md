# Sözleşme şeması

Kontenjan sözleşmelerinden çıkarılacak yapılandırılmış verinin tanımı.

## Dosyalar

| Dosya | İçerik |
|---|---|
| `kontrat.json` | JSON Schema (draft 2020-12) |
| `ornek-argos.json` | MEGEP örnek sözleşmesinin şemaya doldurulmuş hali; test fixture |

## Kaynaklar

Şema kamuya açık kaynaklardan türetildi; hiçbir otelden gerçek sözleşme talep edilmedi.

- **MEB MEGEP, "Paket Tur Üretimi" modülü (812STE003)** — madde madde numaralanmış
  örnek kontenjan sözleşmesi (Side Turizm – Argos Otel), kontenjan kontratı çeşitleri,
  pansiyon kısaltmaları, iptal süreleri, ödeme koşulları.
- **MEB MEGEP, "Rezervasyon İşlemleri" modülü** — kontenjan dışı rezervasyon ve iptal akışı.
- **Turizm İşletmelerinin Bakanlıkla, Birbirleriyle ve Müşterileriyle İlişkileri Hakkında
  Yönetmelik** — üç sözleşme tipinin tanımı ve sözleşmede bulunması gereken hususlar.

MEGEP modülü 2011 tarihlidir ve klasik acente–otel sözleşmesini anlatır. Modern tur
operatörü kontratlarındaki `release` ve `stop_sale` terminolojisi orada geçmez; bu iki
alan sektörün güncel kullanımından eklenmiştir. `release` alanının mevzuat karşılığı,
MEGEP örneğindeki "isim listesini girişten 10 gün önce bildirme" yükümlülüğüdür.

## Kapsam

Beş çekirdek alan zorunludur ve Okuyucu agent yalnızca bunları çıkarır:

1. `donem` — sözleşme dönemi ve alt fiyat dönemleri
2. `oda_kontenjanlari` — oda tipi ve tahsis adedi
3. `fiyatlar` — oda tipi, pansiyon, tutar, birim
4. `release` — bildirim/iade süresi
5. `stop_sale` — satış durdurma aralıkları

`opsiyonel` altındaki alanların şemada yeri vardır ancak agent bunları çıkarmaya
çalışmaz. Kapsam kilidi gereğidir; genişletmeden önce zaman bütçesi kontrol edilmelidir.

## Doğrulama

```bash
pip install jsonschema
python -c "
import json, jsonschema
s = json.load(open('kontrat.json'))
d = json.load(open('ornek-argos.json'))
jsonschema.Draft202012Validator.check_schema(s)
jsonschema.validate(d, s)
print('gecerli')
"
```

## Şemanın yakaladığı tutarsızlık

Argos Otel sözleşmesinde kontenjan tablosundaki oda tipleri (`normal`, `suit`, `balayı`,
`özürlü`) ile fiyat tablosundaki tipler (`tek kişilik`, `iki kişilik`, `üç kişilik`, ...)
örtüşmüyor. Gerçek sözleşmelerde sık görülen bir tutarsızlıktır ve Denetçi agent'ın
(Aşama 8) kurallarından biri olacaktır: *fiyat listesinde karşılığı olmayan oda tipi*.
