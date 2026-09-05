# =====================================================================
# Kontrata — Qwen2.5-1.5B LoRA fine-tune (chunked + full, tek hücre)
#
# ÖNERİLEN EĞİTİM: üretimle hizalı (meta + A/B/C/D parçaları).
# Colab'a tek hücre olarak yapıştırıp çalıştırın.
#
# Bellek: batch 2 + seq 1024 + gradient checkpoint (L4 OOM sonrası).
# Hâlâ OOM → TRAIN_BATCH=1, MAX_SEQ_LEN=768; Runtime'ı yeniden başlat.
#
# ÖN KOŞULLAR
#   1. Çalışma zamanı → GPU (L4 tercih; T4 olur)
#   2. Secrets: HF_TOKEN (Hugging Face yazma yetkisi)
#   3. prompt.go / chunk_prompt.go main'de güncel olmalı (bu hücre
#      GitHub'dan klonlar). Yerel değişiklik varsa önce push edin.
#
# ÇIKTI (v2 — mevcut v1 endpoint'i ezmez; doğruladıktan sonra bağlayın)
#   fatmaoz/kontrata-qwen-lora-v2
#   fatmaoz/kontrata-qwen-merged-v2
#
# Veri: her sözleşme → 5 sohbet örneği
#   FULL  : SYSTEM_PROMPT + tam JSON (single fallback)
#   A/B/C/D: chunk prompt + kısmi JSON (EXTRACT_MODE=chunked)
# =====================================================================

# --------------------------------------------------------------- 0. kurulum
import subprocess, sys, os, json, re, gc, time, shutil
from pathlib import Path

print("=== paketler kuruluyor ===")
subprocess.check_call([
    sys.executable, "-m", "pip", "install", "-q",
    "transformers", "peft", "trl", "bitsandbytes", "accelerate",
    "datasets", "huggingface_hub", "jsonschema", "faker",
])

import torch
print("GPU:", torch.cuda.get_device_name(0) if torch.cuda.is_available() else "YOK")
if not torch.cuda.is_available():
    raise SystemExit("GPU yok. Çalışma zamanı → türünü değiştir → GPU seçin.")

from google.colab import userdata
from huggingface_hub import login, HfApi

HF_TOKEN = userdata.get("HF_TOKEN")
if not HF_TOKEN:
    raise SystemExit("Colab Secrets'a HF_TOKEN ekleyin.")
login(token=HF_TOKEN)
print("Hugging Face oturumu açıldı (jeton yazdırılmadı)")

# --------------------------------------------------------------- 1. sabitler
HF_USER = "fatmaoz"
GITHUB_REPO = "https://github.com/oz-fatma/kontrata.git"
BASE_MODEL = "Qwen/Qwen2.5-1.5B-Instruct"
ADAPTER_DIR = "/content/kontrata-qwen-lora"
MERGED_DIR = "/content/kontrata-qwen-merged"
ADAPTER_REPO = f"{HF_USER}/kontrata-qwen-lora-v2"
MERGED_REPO = f"{HF_USER}/kontrata-qwen-merged-v2"

EPOCHS = 4
N_TRAIN = 320
N_VAL = 80
# 1536 + batch 4 L4'te OOM verdi; 1024 + batch 2 + grad ckpt güvenli
MAX_SEQ_LEN = 1024
MAX_NEW_TOK = 400
N_EVAL = 24  # 6 sözleşme × (full+A/B/C/D) karışımı; aşağıda satır bazlı
TRAIN_BATCH = 2
GRAD_ACCUM = 8  # efektif batch 16
EVAL_BATCH = 2

# Üretimle BİREBİR: backend/internal/agent/prompt.go
SYSTEM_PROMPT = """Sen bir kontenjan sözleşmesi çıkarım motorusun. Verilen sözleşme metninden JSON üret.

SADECE JSON döndür. Tablo, markdown, açıklama YAZMA.

Çıktı tam olarak şu biçimde olmalı:
{"meta":{"otel_adi":"Argos Otel","acente_adi":"Side Turizm","para_birimi":"GBP","kur_esasi":"giris_gunu_tcmb","yetkili_mahkeme":"Antalya"},"donem":{"baslangic":"2026-04-01","bitis":"2026-10-31","alt_donemler":[]},"oda_kontenjanlari":[{"oda_tipi":"standart","adet":170}],"fiyatlar":[{"oda_tipi":"standart","tutar":50,"birim":"oda_gecelik","pansiyon":"belirtilmemis"}],"release":{"gun":10,"kapsam":"isim_listesi"},"stop_sale":[]}

Alan kuralları:
- meta (isteğe bağlı, sadece şu alanlar): otel_adi, acente_adi, para_birimi (EUR|GBP|USD|TRY), kur_esasi (giris_gunu_tcmb|cikis_gunu_tcmb|sabit_kur|belirtilmemis), yetkili_mahkeme (sadece şehir adı), sozlesme_tipi (tamamen_garantili|kismen_garantili|garantisiz|istege_bagli|serbest_satis|blok_rezervasyon|blok_satin_alma|belirtilmemis), sezon (yaz|kis|yillik|belirtilmemis)
- donem.baslangic, donem.bitis: ISO tarih veya null
- oda_kontenjanlari: oda_tipi (standart/suit/balayi/engelli/aile/deluxe), adet (tam sayı)
- fiyatlar: oda_tipi, tutar (sayı), birim (oda_gecelik|kisi_gecelik), pansiyon (RO|BB|HB|FB|AI|belirtilmemis)
- release: gun (tam sayı), kapsam (isim_listesi|kontenjan_iadesi|her_ikisi|belirtilmemis)
- stop_sale: dizi, yoksa []

Metin İngilizce olabilir; çıktı alan adları ve değerleri şemadaki Türkçe biçimde kalır.

Tek JSON nesnesi. Bittiğinde dur.
meta alanı yalnızca en üstte bir kez yazılır. Diğer alanların içine meta bilgisi (yetkili_mahkeme, para_birimi vb.) yazma.
"""

# Üretimle BİREBİR: backend/internal/agent/chunk_prompt.go
CHUNK_PROMPT_A = """Sen bir kontenjan sözleşmesi çıkarım motorusun. Verilen sözleşme metninden JSON üret.

SADECE JSON döndür. Tablo, markdown, açıklama YAZMA.

Çıktın SADECE şu iki üst düzey alanı içerecek: donem, oda_kontenjanlari. BAŞKA HİÇBİR üst düzey alan YASAK — meta, kontenjan_detaları, release_suresi, stop_sale_suresi, fiyatlar gibi hiçbir ek alan ekleme, ne isimle olursa olsun.

donem alanı SADECE şunları içerir: baslangic, bitis, alt_donemler. alt_donemler yalnızca donem nesnesinin içinde yazılır; üst düzeyde alt_donemler alanı YASAK. alt_donemler metinde açıkça birden fazla farklı fiyat dönemi/sezon TABLOSU yoksa boş dizi [] bırak, uydurma veya tarih listesi yazma.

donem.baslangic ve donem.bitis metinde yazıldığı gibi yaz. Tarihler ters görünse bile (bitiş < başlangıç) yer değiştirme, "düzeltme".

oda_kontenjanlari listesi tamamlanınca ve donem alanı da yazılınca HEMEN dur. Fazladan alan, fazladan açıklama, tekrar YAZMA.

Çıktı tam olarak şu biçimde olmalı:
{"donem":{"baslangic":"2026-04-01","bitis":"2026-10-31","alt_donemler":[]},"oda_kontenjanlari":[{"oda_tipi":"standart","adet":170}]}

Alan kuralları:
- donem.baslangic, donem.bitis: ISO tarih veya null
- oda_kontenjanlari: oda_tipi (standart/suit/balayi/engelli/aile/deluxe), adet (tam sayı). Metindeki her oda satırı için bir nesne yaz.
- İngilizce oda tiplerini Türkçeye çevir: standard->standart, family->aile, suite/junior suite/penthouse->suit, honeymoon->balayi, accessible->engelli, single/double/triple->standart.
- adet alanı SADECE metinde yazan tam sayı olsun. Hesap yapma, çarpma/toplama işlemi YAZMA. Örnek: metin '170 normal oda' diyorsa adet=170 yaz, başka bir şey hesaplama.
- Her oda_tipi için EN FAZLA BİR kayıt üret. Aynı oda tipini tekrar etme. Sözleşmede kaç farklı oda tipi geçiyorsa oda_kontenjanlari dizisinde o kadar eleman olsun, daha fazla değil.
- fiyatlar bölümündeki bilgiyi oda_kontenjanlari'na KARIŞTIRMA. Bu parçada SADECE kontenjan (adet) bilgisi istenir, fiyat/tutar bilgisi bu alanda YOKTUR.
- Kontenjan adetlerini yalnızca kontenjan maddesinden (MADDE 2 / ARTICLE 2 / ROOM ALLOTMENT vb.) al. Fiyat maddesindeki (MADDE 8, ARTICLE 4 RATES, Gecelik ücret vb.) satırları oda_kontenjanlari'na ekleme. Fiyat tutarını adet olarak YAZMA.
- Kontenjan bölümündeki cümlede geçen tüm oda tiplerini yaz; balayı, özürlü/engelli dahil hiçbirini atlama. 'normal oda' -> standart, 'özürlü oda' -> engelli, 'balayı odası' -> balayi. engelli ve özürlü aynı tiptir; yalnızca engelli olarak tek kayıt yaz.
- oda_kontenjanlari nesnelerinde yalnızca oda_tipi ve adet yaz; kapsam, tutar, birim, pansiyon ekleme.

Metin İngilizce olabilir; çıktı alan adları ve değerleri şemadaki Türkçe biçimde kalır.

Tek JSON nesnesi. İki alan yazıldıktan sonra dur."""

CHUNK_PROMPT_B = """Sen bir kontenjan sözleşmesi çıkarım motorusun. Verilen sözleşme metninden JSON üret.

SADECE fiyatlar alanını üret, dizi olarak. Başka hiçbir üst alan yazma.

SADECE JSON döndür. Tablo, markdown, açıklama YAZMA.

Çıktı tam olarak şu biçimde olmalı:
{"fiyatlar":[{"oda_tipi":"standart","tutar":50,"birim":"oda_gecelik","pansiyon":"belirtilmemis"}]}

Alan kuralları:
- fiyatlar: oda_tipi, tutar (sayı), birim (oda_gecelik|kisi_gecelik), pansiyon (RO|BB|HB|FB|AI|belirtilmemis)
- İngilizce oda tiplerini Türkçeye çevir: standard->standart, family->aile, suite/junior suite/penthouse->suit, honeymoon->balayi, accessible->engelli
- Fiyat/Rates tablosundaki HER satırı yaz; tek satır bırakma. Junior suite ve penthouse de dahil (ikisi de suit olabilir).
- Tablo sütunları Erken/Yüksek/Geç sezon gibi birden fazla dönem fiyatı içeriyorsa her oda tipi × her dönem için ayrı nesne yaz; alt_donem_ad alanına dönem adını koy (örn. "Erken sezon").
- "kişi başı" / "per person" ise birim=kisi_gecelik; "per room" / oda gecelik ise oda_gecelik.
- Pansiyon: AI/BB/HB/FB/RO metinde varsa yaz; bed and breakfast -> BB; all inclusive / her şey dahil -> AI.
- Kontenjan adetlerini fiyat olarak YAZMA.

Metin İngilizce olabilir; çıktı alan adları ve değerleri şemadaki Türkçe biçimde kalır.

Tek JSON nesnesi. Bittiğinde dur."""

CHUNK_PROMPT_C = """Sen bir kontenjan sözleşmesi çıkarım motorusun. Verilen sözleşme metninden JSON üret.

SADECE release ve stop_sale alanlarını üret.

SADECE JSON döndür. Tablo, markdown, açıklama YAZMA.

Çıktı tam olarak şu biçimde olmalı:
{"release":{"gun":10,"kapsam":"isim_listesi"},"stop_sale":[]}

Alan kuralları:
- release: TEK nesne (dizi değil). gun (tam sayı, "10 gün" yazma — sadece 10), kapsam (isim_listesi|kontenjan_iadesi|her_ikisi|belirtilmemis)
- stop_sale: dizi. Metinde "stop-sale" / "satış durdurma" maddesi veya tablosu YOKSA mutlaka [].
- stop_sale varsa her satır: {"baslangic":"YYYY-MM-DD","bitis":"YYYY-MM-DD","kapsam":"..."}. Kapsam oda tipi veya "tüm oda tipleri" olabilir.
- Sözleşme dönemi tarihlerini stop_sale olarak UYDURMA. Sezon adlarından stop_sale UYDURMA. Yalnızca metinde yazan başlangıç/bitiş tarihli satırları yaz.

Metin İngilizce olabilir; çıktı alan adları ve değerleri şemadaki Türkçe biçimde kalır.

Tek JSON nesnesi. Bittiğinde dur."""

CHUNK_PROMPT_D = """Sen bir kontenjan sözleşmesi çıkarım motorusun. Verilen sözleşme metninden JSON üret.

SADECE meta alanını üret, başka hiçbir üst alan yazma.

SADECE JSON döndür. Tablo, markdown, açıklama YAZMA.

Öncelik sırası (metinde varsa yaz, yoksa atla):
1) otel_adi — tesis/otel adı (Taraflar satırındaki Tesis/Otel/Hotel; veya "X ile Y arasında" içindeki otel)
2) acente_adi — operatör/acente adı (Taraflar satırındaki Operatör/Acente; veya "X ile Y arasında" içindeki acente)
3) para_birimi — EUR|GBP|USD|TRY (fiyat maddesindeki para birimi: EUR, GBP, İngiliz Sterlini, Euro, USD, TRY/TL)
4) yetkili_mahkeme — yalnızca şehir adı (Antalya gibi)
5) kur_esasi — yalnızca metinde kur kuralı AÇIKÇA varsa: giris_gunu_tcmb|cikis_gunu_tcmb|sabit_kur

sozlesme_tipi ve sezon YALNIZCA metinde birebir karşılığı varsa yaz.
- sozlesme_tipi: tamamen_garantili|kismen_garantili|garantisiz|istege_bagli|serbest_satis|blok_rezervasyon|blok_satin_alma
- sezon: yaz|kis|yillik
"Erken sezon / Yüksek sezon / Geç sezon" dönem adları sezon alanı DEĞİLDİR — sezon yazma.
"belirtilmemis" yazma; emin değilsen alanı hiç koyma.
Tahmin etme, uydurma.

Çıktı örneği:
{"meta":{"otel_adi":"Argos Otel","acente_adi":"Side Turizm","para_birimi":"GBP","yetkili_mahkeme":"Antalya","kur_esasi":"giris_gunu_tcmb"}}

Metin İngilizce olabilir; çıktı alan adları ve değerleri şemadaki Türkçe biçimde kalır.

Tek JSON nesnesi. Bittiğinde dur."""

# --------------------------------------------------------------- 2. veri
print("\n=== veri üretiliyor (GitHub'dan) ===")
REPO = Path("/content/kontrata")
if REPO.exists():
    shutil.rmtree(REPO)
subprocess.check_call(["git", "clone", "--depth", "1", GITHUB_REPO, str(REPO)])
subprocess.check_call([
    sys.executable, str(REPO / "ml" / "generate.py"),
    "--seed", "42", "--train", str(N_TRAIN), "--val", str(N_VAL),
])

TRAIN_PATH = REPO / "ml" / "data" / "train.jsonl"
VAL_PATH = REPO / "ml" / "data" / "val.jsonl"
SCHEMA_PATH = REPO / "ml" / "schema" / "kontrat.json"


def load_jsonl(p: Path) -> list:
    return [json.loads(l) for l in p.open(encoding="utf-8") if l.strip()]


def clean_meta(meta) -> dict:
    """Parça D: belirtilmemis / boş alanları düş (üretim kuralı)."""
    if not isinstance(meta, dict):
        return {}
    out = {}
    for k, v in meta.items():
        if v is None or v == "" or v == "belirtilmemis":
            continue
        out[k] = v
    return out


def slim_kontenjan(items) -> list:
    if not isinstance(items, list):
        return []
    out = []
    for x in items:
        if not isinstance(x, dict):
            continue
        row = {"oda_tipi": x.get("oda_tipi"), "adet": x.get("adet")}
        out.append(row)
    return out


def expand_row(row: dict) -> list:
    """Bir sözleşmeden FULL + A/B/C/D eğitim örnekleri."""
    metin = row["metin"]
    cikti = row.get("cikti") or {}
    examples = []

    examples.append({
        "parca": "FULL",
        "messages": [
            {"role": "system", "content": SYSTEM_PROMPT},
            {"role": "user", "content": metin},
            {"role": "assistant", "content": json.dumps(cikti, ensure_ascii=False)},
        ],
    })

    gold_a = {
        "donem": cikti.get("donem"),
        "oda_kontenjanlari": slim_kontenjan(cikti.get("oda_kontenjanlari")),
    }
    examples.append({
        "parca": "A",
        "messages": [
            {"role": "system", "content": CHUNK_PROMPT_A},
            {"role": "user", "content": metin},
            {"role": "assistant", "content": json.dumps(gold_a, ensure_ascii=False)},
        ],
    })

    gold_b = {"fiyatlar": cikti.get("fiyatlar") or []}
    examples.append({
        "parca": "B",
        "messages": [
            {"role": "system", "content": CHUNK_PROMPT_B},
            {"role": "user", "content": metin},
            {"role": "assistant", "content": json.dumps(gold_b, ensure_ascii=False)},
        ],
    })

    release = cikti.get("release")
    if release is None:
        release = {"gun": None, "kapsam": "belirtilmemis"}
    gold_c = {
        "release": release,
        "stop_sale": cikti.get("stop_sale") or [],
    }
    examples.append({
        "parca": "C",
        "messages": [
            {"role": "system", "content": CHUNK_PROMPT_C},
            {"role": "user", "content": metin},
            {"role": "assistant", "content": json.dumps(gold_c, ensure_ascii=False)},
        ],
    })

    gold_d = {"meta": clean_meta(cikti.get("meta"))}
    examples.append({
        "parca": "D",
        "messages": [
            {"role": "system", "content": CHUNK_PROMPT_D},
            {"role": "user", "content": metin},
            {"role": "assistant", "content": json.dumps(gold_d, ensure_ascii=False)},
        ],
    })
    return examples


train_rows = load_jsonl(TRAIN_PATH)
val_rows = load_jsonl(VAL_PATH)

diller = {}
for r in train_rows:
    d = r.get("meta", {}).get("dil", "?")
    diller[d] = diller.get(d, 0) + 1

train_ex = [ex for row in train_rows for ex in expand_row(row)]
val_ex = [ex for row in val_rows for ex in expand_row(row)]
parca_say = {}
for ex in train_ex:
    parca_say[ex["parca"]] = parca_say.get(ex["parca"], 0) + 1

print(f"sözleşme train={len(train_rows)} val={len(val_rows)} dil={diller}")
print(f"örnek   train={len(train_ex)} val={len(val_ex)} parça={parca_say}")

from datasets import Dataset

train_ds = Dataset.from_list([{"messages": e["messages"]} for e in train_ex])
val_ds = Dataset.from_list([{"messages": e["messages"]} for e in val_ex])

# --------------------------------------------------------------- 3. model
print("\n=== model yükleniyor ===")
from peft import LoraConfig
from transformers import AutoModelForCausalLM, AutoTokenizer, BitsAndBytesConfig

compute_dtype = torch.bfloat16 if torch.cuda.is_bf16_supported() else torch.float16

bnb = BitsAndBytesConfig(
    load_in_4bit=True,
    bnb_4bit_quant_type="nf4",
    bnb_4bit_use_double_quant=True,
    bnb_4bit_compute_dtype=compute_dtype,
)

tokenizer = AutoTokenizer.from_pretrained(BASE_MODEL, trust_remote_code=True)
if tokenizer.pad_token is None:
    tokenizer.pad_token = tokenizer.eos_token
tokenizer.padding_side = "right"

model = AutoModelForCausalLM.from_pretrained(
    BASE_MODEL, quantization_config=bnb, device_map="auto", trust_remote_code=True,
)
model.config.use_cache = False
# LoRA + uzun sohbet için aktivasyon bellek tasarrufu
if hasattr(model, "gradient_checkpointing_enable"):
    model.gradient_checkpointing_enable()
    if hasattr(model, "enable_input_require_grads"):
        model.enable_input_require_grads()

peft_config = LoraConfig(
    r=16, lora_alpha=32, lora_dropout=0.05, bias="none", task_type="CAUSAL_LM",
    target_modules=[
        "q_proj", "k_proj", "v_proj", "o_proj",
        "gate_proj", "up_proj", "down_proj",
    ],
)
print(f"taban={BASE_MODEL} dtype={compute_dtype} LoRA r=16 alpha=32")

# --------------------------------------------------------------- 4. eğitim
print("\n=== eğitim başlıyor ===")
from trl import SFTConfig, SFTTrainer

sft_args = SFTConfig(
    output_dir=ADAPTER_DIR,
    num_train_epochs=EPOCHS,
    per_device_train_batch_size=TRAIN_BATCH,
    per_device_eval_batch_size=EVAL_BATCH,
    gradient_accumulation_steps=GRAD_ACCUM,
    learning_rate=2e-4,
    lr_scheduler_type="cosine",
    warmup_steps=10,
    logging_steps=10,
    eval_strategy="epoch",
    save_strategy="epoch",
    max_length=MAX_SEQ_LEN,
    report_to="none",
    optim="paged_adamw_8bit",
    gradient_checkpointing=True,
    gradient_checkpointing_kwargs={"use_reentrant": False},
    bf16=compute_dtype == torch.bfloat16,
    fp16=compute_dtype != torch.bfloat16,
)

trainer = SFTTrainer(
    model=model,
    args=sft_args,
    train_dataset=train_ds,
    eval_dataset=val_ds,
    peft_config=peft_config,
    processing_class=tokenizer,
)

gc.collect()
torch.cuda.empty_cache()
print(
    f"eğitim ayarı: batch={TRAIN_BATCH} accum={GRAD_ACCUM} "
    f"seq={MAX_SEQ_LEN} grad_ckpt=on  "
    f"boş VRAM≈{torch.cuda.mem_get_info()[0]/1024**3:.1f} GiB"
)

t0 = time.perf_counter()
train_result = trainer.train()
elapsed = time.perf_counter() - t0
trainer.save_model(ADAPTER_DIR)
tokenizer.save_pretrained(ADAPTER_DIR)

print(f"\neğitim süresi: {elapsed/60:.1f} dakika")
for h in trainer.state.log_history:
    if "eval_loss" in h:
        print(f"  epoch={h.get('epoch'):.0f}  val_loss={h['eval_loss']:.4f}")

# --------------------------------------------------------- 5. adapter -> HF
print("\n=== adapter HF'ye gönderiliyor ===")
ADAPTER_CARD = f"""---
base_model: {BASE_MODEL}
library_name: peft
license: apache-2.0
tags: [lora, qwen2.5, text-generation, kontrata]
---

# kontrata-qwen-lora-v2

Kontenjan sözleşmesi → şema JSON. **Chunked + full** SFT.

Endpoint için birleşik: `{MERGED_REPO}`.

## Eğitim
- Taban: `{BASE_MODEL}`, 4-bit nf4
- LoRA r=16, alpha=32, dropout=0.05
- {len(train_rows)} sözleşme → {len(train_ex)} sohbet örneği (FULL+A/B/C/D)
- Dil: {diller}
- {EPOCHS} epoch, max_length={MAX_SEQ_LEN}
- Promptlar: `prompt.go` + `chunk_prompt.go` ile hizalı

## Kullanım
- Üretim: `EXTRACT_MODE=chunked` (önerilen)
- Single fallback: FULL prompt ile eğitildi

## Sınırlamalar
- Sentetik veri; gerçek operatör PDF'leri ayrı ölçülmeli
- Ham çıktı `backend/internal/extract` onarımından geçmeli
"""
Path(ADAPTER_DIR, "README.md").write_text(ADAPTER_CARD, encoding="utf-8")

api = HfApi()
api.create_repo(ADAPTER_REPO, exist_ok=True, private=True)
api.upload_folder(folder_path=ADAPTER_DIR, repo_id=ADAPTER_REPO)
print("yüklendi:", ADAPTER_REPO)

# --------------------------------------------------------- 6. değerlendirme
print("\n=== değerlendirme (parça karışımı) ===")
from jsonschema import Draft202012Validator, FormatChecker

schema = json.loads(SCHEMA_PATH.read_text(encoding="utf-8"))
validator = Draft202012Validator(schema, format_checker=FormatChecker())


def extract_json(text: str):
    if not text or not text.strip():
        return None
    raw = text.strip()
    fence = re.search(r"```(?:json)?\s*([\s\S]*?)```", raw, re.IGNORECASE)
    if fence:
        raw = fence.group(1).strip()
    dec = json.JSONDecoder()
    birlesik, i, n = {}, 0, len(raw)
    while i < n:
        while i < n and raw[i] != "{":
            i += 1
        if i >= n:
            break
        try:
            obj, son = dec.raw_decode(raw, i)
        except json.JSONDecodeError:
            i += 1
            continue
        if isinstance(obj, dict):
            birlesik.update(obj)
        i = son
    return birlesik or None


model.eval()
model.config.use_cache = True
device = next(model.parameters()).device

eos_ids = [tokenizer.eos_token_id]
im_end = tokenizer.convert_tokens_to_ids("<|im_end|>")
if isinstance(im_end, int) and im_end >= 0 and im_end not in eos_ids:
    eos_ids.append(im_end)

# İlk birkaç val sözleşmesinden her parçayı bir kez dene
eval_items = []
for row in val_rows[: max(1, N_EVAL // 5)]:
    eval_items.extend(expand_row(row))

kayitlar = []
for i, item in enumerate(eval_items):
    msgs = item["messages"][:2]  # system + user
    gold = json.loads(item["messages"][2]["content"])
    p = tokenizer.apply_chat_template(
        msgs, tokenize=False, add_generation_prompt=True,
    )
    inp = tokenizer(p, return_tensors="pt").to(device)
    with torch.no_grad():
        out = model.generate(
            **inp,
            max_new_tokens=MAX_NEW_TOK,
            do_sample=False,
            eos_token_id=eos_ids,
            pad_token_id=tokenizer.pad_token_id,
        )
    gen = tokenizer.decode(out[0][inp["input_ids"].shape[1]:], skip_special_tokens=True)
    parsed = extract_json(gen)
    ok_json = isinstance(parsed, dict)
    # FULL için şema; parçalar için yalnızca anahtar örtüşmesi
    if item["parca"] == "FULL":
        ok_sema = ok_json and not any(validator.iter_errors(parsed))
    else:
        ok_sema = ok_json and set(gold.keys()).issubset(set(parsed.keys()))
    kayitlar.append({
        "parca": item["parca"],
        "json": ok_json,
        "sema": ok_sema,
    })
    print(f"  {i+1}/{len(eval_items)} parça={item['parca']} json={ok_json} alan={ok_sema}")


def oran(vals):
    return 100 * sum(1 for v in vals if v) / len(vals) if vals else 0.0


print("\n--- genel ---")
print(f"  geçerli JSON : {oran([k['json'] for k in kayitlar]):5.1f}%")
print(f"  alan/şema    : {oran([k['sema'] for k in kayitlar]):5.1f}%")
for parca in ("FULL", "A", "B", "C", "D"):
    alt = [k for k in kayitlar if k["parca"] == parca]
    if alt:
        print(
            f"  {parca:4s} n={len(alt)}  json={oran([k['json'] for k in alt]):5.1f}%  "
            f"ok={oran([k['sema'] for k in alt]):5.1f}%"
        )

# --------------------------------------------------------- 7. merge -> HF
print("\n=== adapter birleştiriliyor ve gönderiliyor ===")
# peft merge bazen torchao>=0.16 ister; Colab'da eski sürüm kalabiliyor
subprocess.check_call([
    sys.executable, "-m", "pip", "install", "-q", "-U", "torchao>=0.16.0", "peft",
])
from peft import PeftModel

del model, trainer
gc.collect()
torch.cuda.empty_cache()

base = AutoModelForCausalLM.from_pretrained(
    BASE_MODEL, torch_dtype=torch.bfloat16, device_map="cpu", trust_remote_code=True,
)
merged = PeftModel.from_pretrained(base, ADAPTER_DIR).merge_and_unload()

Path(MERGED_DIR).mkdir(parents=True, exist_ok=True)
merged.save_pretrained(MERGED_DIR)
tokenizer.save_pretrained(MERGED_DIR)

tc = Path(MERGED_DIR, "tokenizer_config.json")
cfg = json.loads(tc.read_text(encoding="utf-8"))
if isinstance(cfg.get("extra_special_tokens"), list):
    cfg["extra_special_tokens"] = {}
    tc.write_text(json.dumps(cfg, ensure_ascii=False, indent=2), encoding="utf-8")
    print("tokenizer_config düzeltildi (extra_special_tokens -> {})")

Path(MERGED_DIR, "README.md").write_text(f"""---
base_model: {BASE_MODEL}
license: apache-2.0
tags: [qwen2.5, text-generation, kontrata]
---

# kontrata-qwen-merged-v2

`{ADAPTER_REPO}` birleşik ağırlık. HF Inference Endpoint için bu repo.

Chunked üretim: `EXTRACT_MODE=chunked`.
""", encoding="utf-8")

api.create_repo(MERGED_REPO, exist_ok=True, private=True)
api.upload_folder(folder_path=MERGED_DIR, repo_id=MERGED_REPO)
print("yüklendi:", MERGED_REPO)

print("\n=== TAMAM ===")
print(f"adapter : {ADAPTER_REPO}")
print(f"merged  : {MERGED_REPO}")
print("Sonraki: endpoint'i v2 merged'e bağla; EXTRACT_MODE=chunked ile TUI/Coral/Anex dene.")
print("Eski v1 endpoint'i doğrulayana kadar tut.")
