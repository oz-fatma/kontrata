
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

