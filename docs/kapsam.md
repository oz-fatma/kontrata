
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

# Egitim / uretim prompt farki (zaman kalirsa)

Model, egitimdeki SYSTEM_PROMPT ile uretimde kullanilan prompt farkli.
Gercek PDF metinlerinde egitim promptu markdown tablo ciktisi
tetikliyordu; ornek JSON iceren kisa prompt bunu duzeltti.

Dogru cozum: sentetik veriyi yeniden uretip egitim promptunu
uretimdekiyle esitlemek ve modeli yeniden egitmek (~1.5 saat).
Ayrica egitim verisine gercek PDF cikarimina benzeyen metinler
eklenmeli (tablo hizalamasi, sayfa gecisleri).

Mevcut durum kabul edilebilir: model degerleri dogru cikariyor,
sorun yalnizca cikti bicimiydi ve prompt ile cozuldu.
