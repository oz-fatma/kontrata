
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
Üretim kısa örnek-JSON prompt'una geçti; `ml/train_colab.ipynb` hücre 2,
`ml/evaluate.py` ve `ml/colab_train_chunked.py` artık `backend/internal/agent`
ile birebir aynı metni kullanır (`prompt_test.go` doğrular).
Sentetik veri `%50` TR / `%50` EN, İngilizce şablonlar operatör kontratı diline
çekildi. Modelin yeniden eğitilmesi bu hizalamayı kalıcı kılar.

2026-09-03: üretim prompt'una isteğe bağlı `meta` (otel_adi, acente_adi,
para_birimi, kur_esasi, yetkili_mahkeme, sozlesme_tipi, sezon) eklendi.
`sozlesme_tipi` ve `sezon` geçerli değerleri, `yetkili_mahkeme` için şehir
adı kısıtı ve “meta yalnızca kökte bir kez” kuralı prompt'ta yazılıdır.
Notebook, `evaluate.py` ve chunked Colab script aynı metni taşır.


# Model yukseltmesi (Asama 11 sonrasi)

Durum: ertelendi (2026-09-05). Önce 3B denendi; val kaybı düştü ama
üretimde JSON parçalanması ve T4 gecikmesi nedeniyle vazgeçildi.
Asıl çözüm olarak **bölümsel çıkarım** uygulandı (aşağıdaki bölüm).

Üretim: `Qwen2.5-1.5B` + `…-merged-v1` + `EXTRACT_MODE=chunked`.
Daha büyük taban (3B+) ileride ayrı karar; şu an öncelik değil.

# Bölümsel çıkarım (chunking)

Durum: uygulandı (2026-09-04). Karar: `docs/kararlar.md` §30.

`EXTRACT_MODE=chunked` ile Okuyucu dört paralel çağrı yapar:
1. A — donem + oda_kontenjanlari
2. B — fiyatlar
3. C — release + stop_sale
4. D — meta (ayrı; A'ya eklenince model bozuluyordu)

Birleşim sonrası mevcut Normalize + Validate + Denetçi hattı çalışır.
Varsayılan hâlâ `single`; üretimde `chunked` önerilir.
Chunked SFT deneyi: `ml/colab_train_chunked.py` → `…-merged-v2` (rafta).
Üretim endpoint'i şimdilik `…-merged-v1` + chunked çıkarım.

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

Sonraki adim uygulandi: bolumsel cikarim (yukaridaki bolum + karar §30).
Uretim: 1.5B (`…-merged-v1`) + `EXTRACT_MODE=chunked`. Chunked-egitilmis
v2 agirligi Hub'da duruyor; zor PDF'lerde v1+chunked simdilik daha guvenilir.
