
# Ölçekleme gereksinimi (Aşama 11)

İki LLM ucu ve aralarında yüke göre dağıtım:
- Uç 1: HF Inference Endpoint (fine-tune edilmiş model)
- Uç 2: Yerel Ollama (aynı model)
- Go tarafında yönlendirici: kuyruk derinliği, gecikme, saglik durumuna
  gore uc secimi; bir uc duserse digerine kayma
- 5 eszamanli kullanici senaryosu: ikinci uc devreye girmeli
- Kanit: yuk testi ve uc bazinda dagilim/gecikme tablosu (README)

Acik soru: "2 LLM" iki farkli model mi, ayni modelin iki ornegi mi?
Cevaba gore mimari degisir. Asama 6'dan once netlesmeli.

Butce: Asama 11 5 -> 11 saat.
Kesintiler: Asama 5 (5->4), Asama 9 (10->7), Asama 10 (6->5),
Asama 12 (5->4).

# Eğitim / üretim prompt farkı

Durum: çözüldü (2026-09-03).

Eğitimdeki uzun SYSTEM_PROMPT gerçek PDF'lerde markdown tablo tetikliyordu.
Üretim kısa örnek-JSON prompt'una geçti; `ml/train_colab.ipynb` hücre 2 ve
`ml/evaluate.py` artık `backend/internal/agent` ile birebir aynı metni kullanır.
Sentetik veri `%50` TR / `%50` EN, İngilizce şablonlar operatör kontratı diline
çekildi. Modelin yeniden eğitilmesi bu hizalamayı kalıcı kılar.

2026-09-03: üretim prompt'una isteğe bağlı `meta` (otel_adi, acente_adi,
para_birimi, kur_esasi, yetkili_mahkeme, sozlesme_tipi, sezon) eklendi.
`sozlesme_tipi` ve `sezon` geçerli değerleri, `yetkili_mahkeme` için şehir
adı kısıtı ve “meta yalnızca kökte bir kez” kuralı prompt'ta yazılıdır.
Notebook ve `evaluate.py` aynı metni taşır; mevcut model yeniden
eğitilmedi — meta opsiyonel olduğu için tarifi izlemesi beklenir.


# Model yukseltmesi (Asama 11 sonrasi)

Qwen2.5-1.5B uzun ve karmasik sozlesmelerde JSON yapisini tutamiyor:
nesneyi parcaliyor, alanlari atliyor, bazen sozdizimi bozuk cikti
veriyor. Ayni PDF ayni ayarlarla farkli sonuc verebiliyor.

Karar: Qwen2.5-3B-Instruct'a gecilecek. Ayni aile, ayni tokenizer,
ayni chat sablonu; notebook'ta tek satir degisiyor.

Adimlar:
1. Colab'da BASE_MODEL degistirip yeniden egit (~25 dk)
2. HF'ye adapter + merged yukle
3. Iki endpoint'i de sil ve yeniden kur (HF model guncellemesini
   otomatik cekmiyor)
4. Tekrarlanabilirlik olcumu: ayni sozlesme 10 kez islenip basari
   orani raporlanacak (Asama 11 izleme katmani ile)

Beklenen etki: uzun girdide yapi tutma belirgin iyilesir, cikarim
suresi ~12 sn'den ~20-25 sn'ye cikar.

# Bölümsel çıkarım (chunking) — gelecek iş

Durum: planlandi, uygulanmadi (2026-09-03).

Karmasik ve cok sayfali sozlesmelerde (or. iki sayfa, cok satirli fiyat
tablosu, alt donemler) Okuyucu tek LLM cagrisinda JSON yapisini tutamiyor.
Duzeltme turu artirmak ve token siniri yukseltmek tek basina yeterli
olmadi.

Planlanan yaklasim: PDF metnini mantiksal bolumlere ayirip ayri cikarim
cagrilari yapmak, sonra birlestirmek. Ornek parcalar:
1. donem + oda_kontenjanlari + meta
2. fiyatlar (alt_donem_ad dahil)
3. release + stop_sale

Maliyet: ~3x LLM cagrisi. Merge sonrasi mevcut Normalize + Validate +
Denetci kurallari tutarliligi kontrol eder.

Oncelik: teslim sonrasi; 1.5B + mevcut onarim hattinda denenecek.

# Model kararsizligi - kok neden analizi ve gelecek plan

Bugun denenenler ve sonuclari:
- Qwen2.5-3B + 800 ornek: val kaybi dustu (0.189) ama uretimde
  JSON 4-32 parcaya bolunuyor, T4'te 20-90 sn surdu. Vazgecildi.
- Duzeltme turu 1->2 + 2. turda temperature=0.2: coral ve tui
  yine basarisiz kaldi. Ucuz bir sigorta olarak tutuldu, ana
  cozum degil.
- Token siniri 600->1500: sorunu cozmedi, model zaten kendi
  karariyla kisa kesiyordu.

Kok neden (uc farkli hata profili var):
- argos-megep: basit, tek sayfa -> zaten calisiyor
- coral-bozuk: model bazen JSON parse edilemeyen/dil karisik
  metin uretiyor
- tui-2026-yaz: karmasik tablo (12 fiyat satiri + alt donemler +
  kontenjan) tek JSON'da tutarli uretilemiyor, yapisal olarak
  eksik kaliyor

Sonraki adim (zaman kalirsa): bolumsel cikarim (chunking).
Sozlesmeyi 3 parcaya bolup ayri LLM cagrisi yapmak:
  1. Meta + donem + kontenjan
  2. Fiyatlar (en buyuk parca)
  3. Release + stop_sale
Sonra merge edip mevcut Normalize/Validate/Denetci hattina sokmak.
Maliyet ~3x LLM cagrisi (~18-36 sn 1.5B+L4 ile tahmini). Bu,
retry/siracklik artirmaktan cok daha yuksek basari sansi tasiyor
cunku kucuk modelin tek seferde tasimasi gereken bilissel yuku
azaltiyor.

Karar: 1.5B model + orijinal prompt (hafif coral/tui iyilestirmesi
ile) uretimde kaliyor. Chunking mimarisi tasarlandi ama zaman
kisitindan uygulanmadi.
