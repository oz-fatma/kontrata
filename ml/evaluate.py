"""Inference Endpoint üzerindeki modeli val.jsonl ve MEGEP Argos örneğiyle ölçer.

Sözleşme gövdesi loga yazılmaz; raporda yalnızca metrikler ve alan eşleşme bayrakları vardır.
"""

from __future__ import annotations

import argparse
import json
import os
import re
import sys
import urllib.error
import urllib.request
from concurrent.futures import ThreadPoolExecutor, as_completed
from datetime import datetime, timezone
from pathlib import Path

from jsonschema import Draft202012Validator, FormatChecker

ROOT = Path(__file__).resolve().parent
SCHEMA_PATH = ROOT / "schema" / "kontrat.json"
ARGOS_PATH = ROOT / "schema" / "ornek-argos.json"
RESULTS_DIR = ROOT / "results"

CORE_FIELDS = ("donem", "oda_kontenjanlari", "fiyatlar", "release", "stop_sale")

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

FIYAT_LABEL_TR = {
    "tek_kisilik": "tek kişilik",
    "iki_kisilik": "iki kişilik",
    "uc_kisilik": "üç kişilik",
    "suit": "suit",
    "balayi": "balayı",
    "engelli": "özürlü odası",
    "standart": "normal oda",
    "aile": "aile",
    "deluxe": "deluxe",
}


def load_jsonl(path: Path) -> list[dict]:
    rows: list[dict] = []
    with path.open(encoding="utf-8") as f:
        for i, line in enumerate(f, 1):
            line = line.strip()
            if not line:
                continue
            try:
                rows.append(json.loads(line))
            except json.JSONDecodeError as e:
                raise SystemExit(f"{path}:{i} JSON okunamadı: {e}") from e
    return rows


def extract_json(text: str) -> object | None:
    """Model çıktısından ilk JSON nesnesini ayıklar; parse edilemezse None."""
    if not text or not text.strip():
        return None
    raw = text.strip()
    fence = re.search(r"```(?:json)?\s*([\s\S]*?)```", raw, re.IGNORECASE)
    if fence:
        raw = fence.group(1).strip()
    start = raw.find("{")
    end = raw.rfind("}")
    if start == -1 or end == -1 or end <= start:
        return None
    try:
        return json.loads(raw[start : end + 1])
    except json.JSONDecodeError:
        return None


def _sort_key(item: dict) -> str:
    return json.dumps(item, sort_keys=True, ensure_ascii=False)


def canonicalize_field(name: str, value: object) -> object:
    """Açıklama / kaynak_ifade dışındaki çekirdek değeri karşılaştırılabilir hale getirir."""
    if value is None:
        return None
    if name == "donem":
        if not isinstance(value, dict):
            return value
        alts = value.get("alt_donemler") or []
        canon_alts = []
        if isinstance(alts, list):
            for a in alts:
                if not isinstance(a, dict):
                    continue
                canon_alts.append(
                    {
                        "ad": a.get("ad"),
                        "baslangic": a.get("baslangic"),
                        "bitis": a.get("bitis"),
                    }
                )
            canon_alts.sort(key=_sort_key)
        return {
            "baslangic": value.get("baslangic"),
            "bitis": value.get("bitis"),
            "alt_donemler": canon_alts,
        }
    if name == "oda_kontenjanlari":
        if not isinstance(value, list):
            return value
        rows = []
        for x in value:
            if not isinstance(x, dict):
                continue
            rows.append({"oda_tipi": x.get("oda_tipi"), "adet": x.get("adet")})
        rows.sort(key=_sort_key)
        return rows
    if name == "fiyatlar":
        if not isinstance(value, list):
            return value
        rows = []
        for x in value:
            if not isinstance(x, dict):
                continue
            tutar = x.get("tutar")
            if isinstance(tutar, (int, float)):
                tutar = float(tutar)
            rows.append(
                {
                    "oda_tipi": x.get("oda_tipi"),
                    "tutar": tutar,
                    "birim": x.get("birim"),
                    "pansiyon": x.get("pansiyon") or "belirtilmemis",
                    "alt_donem_ad": x.get("alt_donem_ad"),
                }
            )
        rows.sort(key=_sort_key)
        return rows
    if name == "release":
        if not isinstance(value, dict):
            return value
        return {
            "gun": value.get("gun"),
            "kapsam": value.get("kapsam") or "belirtilmemis",
        }
    if name == "stop_sale":
        if not isinstance(value, list):
            return value
        rows = []
        for x in value:
            if not isinstance(x, dict):
                continue
            rows.append(
                {
                    "baslangic": x.get("baslangic"),
                    "bitis": x.get("bitis"),
                    "kapsam": x.get("kapsam"),
                    "bildirim_yontemi": x.get("bildirim_yontemi") or "belirtilmemis",
                }
            )
        rows.sort(key=_sort_key)
        return rows
    return value


def field_matches(gold: dict | None, pred: dict | None) -> dict[str, bool]:
    out: dict[str, bool] = {}
    gold_obj = gold if isinstance(gold, dict) else {}
    pred_obj = pred if isinstance(pred, dict) else {}
    for name in CORE_FIELDS:
        out[name] = canonicalize_field(name, gold_obj.get(name)) == canonicalize_field(
            name, pred_obj.get(name)
        )
    return out


def schema_ok(obj: object, validator: Draft202012Validator) -> bool:
    if not isinstance(obj, dict):
        return False
    return not any(validator.iter_errors(obj))


def chat_completions_url(endpoint: str) -> str:
    base = endpoint.rstrip("/")
    if base.endswith("/chat/completions"):
        return base
    if base.endswith("/v1"):
        return base + "/chat/completions"
    return base + "/v1/chat/completions"


def tgi_url(endpoint: str) -> str:
    base = endpoint.rstrip("/")
    if base.endswith("/generate"):
        return base
    return base + "/generate"


def post_json(url: str, token: str, payload: dict, timeout: float) -> dict:
    body = json.dumps(payload).encode("utf-8")
    req = urllib.request.Request(
        url,
        data=body,
        headers={
            "Authorization": f"Bearer {token}",
            "Content-Type": "application/json",
            "Accept": "application/json",
        },
        method="POST",
    )
    with urllib.request.urlopen(req, timeout=timeout) as resp:
        return json.loads(resp.read().decode("utf-8"))


def call_endpoint(
    endpoint: str,
    token: str,
    contract_text: str,
    api: str,
    max_tokens: int,
    timeout: float,
) -> str:
    messages = [
        {"role": "system", "content": SYSTEM_PROMPT},
        {"role": "user", "content": contract_text},
    ]
    if api == "chat":
        data = post_json(
            chat_completions_url(endpoint),
            token,
            {
                "model": "tgi",
                "messages": messages,
                "max_tokens": max_tokens,
                "temperature": 0.0,
            },
            timeout,
        )
        return str(data["choices"][0]["message"]["content"])
    prompt = (
        "<|im_start|>system\n"
        f"{SYSTEM_PROMPT}<|im_end|>\n"
        "<|im_start|>user\n"
        f"{contract_text}<|im_end|>\n"
        "<|im_start|>assistant\n"
    )
    data = post_json(
        tgi_url(endpoint),
        token,
        {
            "inputs": prompt,
            "parameters": {
                "max_new_tokens": max_tokens,
                "temperature": 0.0,
                "do_sample": False,
                "return_full_text": False,
            },
        },
        timeout,
    )
    if isinstance(data, list) and data:
        data = data[0]
    if isinstance(data, dict):
        if "generated_text" in data:
            return str(data["generated_text"])
        if "content" in data:
            return str(data["content"])
    raise RuntimeError("TGI yanıtı beklenen biçimde değil")


def argos_source_text(gold: dict) -> str:
    """Altın etiketten MEGEP tarzı düz yazı üretir; ders kitabı metni kopyalanmaz."""
    meta = gold.get("meta") or {}
    acente = meta.get("acente_adi") or "Side Turizm"
    otel = meta.get("otel_adi") or "Argos Otel"
    para = meta.get("para_birimi") or "GBP"
    lines = [
        "KONTENJAN SÖZLEŞMESİ",
        f"{acente} – {otel}",
        "",
        "MADDE 1 — SÖZLEŞME SÜRESİ",
        "Sözleşme süresi bu belgede tarih olarak belirtilmemiştir.",
        "",
        "MADDE 2 — ODA KONTENJANI",
        "Otelin acenteye tahsis ettiği kontenjan:",
    ]
    for room in gold.get("oda_kontenjanlari") or []:
        note = room.get("aciklama") or ""
        if "normal oda" in note:
            label = "normal oda"
        elif "özürlü" in note:
            label = "özürlü odası"
        else:
            label = FIYAT_LABEL_TR.get(room.get("oda_tipi", ""), room.get("oda_tipi", ""))
        lines.append(f"- {label}: {room.get('adet')} adet")
    release = gold.get("release") or {}
    kaynak = release.get("kaynak_ifade") or (
        f"Acente, oda kontenjanı ile ilgili son durumu müşterilerin otele girişini "
        f"{release.get('gun', 10)} gün önce belirler ve isim listesi ile birlikte otele bildirir."
    )
    lines.extend(["", "MADDE 3 — İSİM LİSTESİ", kaynak])
    no_show = (gold.get("opsiyonel") or {}).get("no_show") or {}
    if no_show.get("tazminat_aciklama"):
        lines.extend(["", "MADDE 4 — NO SHOW", no_show["tazminat_aciklama"]])
    overbooking = (gold.get("opsiyonel") or {}).get("overbooking") or {}
    if overbooking.get("aciklama"):
        lines.extend(["", "MADDE 5 — KAPASİTE AŞIMI", overbooking["aciklama"]])
    kur = meta.get("kur_esasi")
    kur_cumle = "Faturalamada döviz kuru esas alınır."
    if kur == "giris_gunu_tcmb":
        kur_cumle = (
            "Faturalamada müşterinin otele giriş günündeki T.C. Merkez Bankası "
            "döviz satış kuru esas alınır."
        )
    lines.extend(
        [
            "",
            "MADDE 6 — ÖDEME VE KUR",
            f"Fiyatlar {para} cinsindendir. {kur_cumle}",
        ]
    )
    mahkeme = meta.get("yetkili_mahkeme")
    if mahkeme:
        lines.extend(
            [
                "",
                "MADDE 7 — YETKİLİ MAHKEME",
                f"İşbu sözleşmeden doğacak uyuşmazlıklarda {mahkeme} mahkemeleri yetkilidir.",
            ]
        )
    lines.extend(["", "MADDE 8 — FİYATLAR", f"Oda fiyatları (oda / gece, {para}):"])
    for price in gold.get("fiyatlar") or []:
        label = FIYAT_LABEL_TR.get(price.get("oda_tipi", ""), price.get("oda_tipi", ""))
        tutar = price.get("tutar")
        lines.append(f"- {label}: {tutar} {para}")
    return "\n".join(lines) + "\n"


def score_prediction(
    gold: dict | None, raw_text: str, validator: Draft202012Validator
) -> dict:
    parsed = extract_json(raw_text)
    valid_json = isinstance(parsed, dict)
    pred = parsed if valid_json else None
    matches = field_matches(gold, pred)
    return {
        "valid_json": valid_json,
        "schema_ok": schema_ok(pred, validator) if valid_json else False,
        "field_match": matches,
        "error": None,
    }


def rate(values: list[bool]) -> float:
    if not values:
        return 0.0
    return sum(1 for v in values if v) / len(values)


def summarize(records: list[dict]) -> dict:
    n = len(records)
    return {
        "n": n,
        "valid_json_rate": rate([r["valid_json"] for r in records]),
        "schema_ok_rate": rate([r["schema_ok"] for r in records]),
        "field_accuracy": {
            name: rate([r["field_match"][name] for r in records]) for name in CORE_FIELDS
        },
        "errors": sum(1 for r in records if r.get("error")),
    }


def pct(x: float) -> str:
    return f"{100.0 * x:5.1f}%"


def print_table(blocks: list[tuple[str, dict]]) -> None:
    headers = ["küme", "n", "geçerli JSON", "şema uyum", *CORE_FIELDS]
    rows = [headers]
    for label, summary in blocks:
        fa = summary["field_accuracy"]
        rows.append(
            [
                label,
                str(summary["n"]),
                pct(summary["valid_json_rate"]),
                pct(summary["schema_ok_rate"]),
                *[pct(fa[name]) for name in CORE_FIELDS],
            ]
        )
    widths = [max(len(r[i]) for r in rows) for i in range(len(headers))]
    for i, row in enumerate(rows):
        line = "  ".join(cell.ljust(widths[j]) for j, cell in enumerate(row))
        print(line)
        if i == 0:
            print("  ".join("-" * w for w in widths))


def eval_one(
    index: int,
    gold: dict,
    contract_text: str,
    endpoint: str,
    token: str,
    api: str,
    max_tokens: int,
    timeout: float,
    validator: Draft202012Validator,
) -> dict:
    try:
        raw = call_endpoint(endpoint, token, contract_text, api, max_tokens, timeout)
        scored = score_prediction(gold, raw, validator)
        scored["index"] = index
        return scored
    except urllib.error.HTTPError as e:
        return {
            "index": index,
            "valid_json": False,
            "schema_ok": False,
            "field_match": {name: False for name in CORE_FIELDS},
            "error": f"HTTP {e.code}",
        }
    except Exception as e:
        return {
            "index": index,
            "valid_json": False,
            "schema_ok": False,
            "field_match": {name: False for name in CORE_FIELDS},
            "error": type(e).__name__,
        }


def run_pool(
    jobs: list[tuple[int, dict, str]],
    endpoint: str,
    token: str,
    api: str,
    max_tokens: int,
    timeout: float,
    validator: Draft202012Validator,
    concurrency: int,
) -> list[dict]:
    results: list[dict] = [{}] * len(jobs)
    with ThreadPoolExecutor(max_workers=max(1, concurrency)) as pool:
        futures = {
            pool.submit(
                eval_one,
                idx,
                gold,
                text,
                endpoint,
                token,
                api,
                max_tokens,
                timeout,
                validator,
            ): pos
            for pos, (idx, gold, text) in enumerate(jobs)
        }
        done = 0
        for fut in as_completed(futures):
            pos = futures[fut]
            results[pos] = fut.result()
            done += 1
            print(f"  {done}/{len(jobs)}", file=sys.stderr)
    return results


def main() -> None:
    parser = argparse.ArgumentParser(
        description="HF Inference Endpoint üzerinde çıkarım metrikleri hesaplar."
    )
    parser.add_argument("--endpoint", required=True, help="Inference Endpoint taban URL'si")
    parser.add_argument(
        "--token",
        default=os.environ.get("HF_TOKEN", ""),
        help="HF jetonu (veya HF_TOKEN ortam değişkeni)",
    )
    parser.add_argument(
        "--data",
        type=Path,
        default=ROOT / "data" / "val.jsonl",
        help="Doğrulama jsonl yolu",
    )
    parser.add_argument("--schema", type=Path, default=SCHEMA_PATH)
    parser.add_argument("--argos", type=Path, default=ARGOS_PATH)
    parser.add_argument("--concurrency", type=int, default=4)
    parser.add_argument("--api", choices=("chat", "tgi"), default="chat")
    parser.add_argument("--max-tokens", type=int, default=2048)
    parser.add_argument("--timeout", type=float, default=180.0)
    parser.add_argument("--out-dir", type=Path, default=RESULTS_DIR)
    args = parser.parse_args()

    if not args.token:
        raise SystemExit("--token veya HF_TOKEN gerekli")
    if not args.data.exists():
        raise SystemExit(f"{args.data} yok; önce python generate.py çalıştırın")
    if not args.schema.exists():
        raise SystemExit(f"{args.schema} bulunamadı")
    if not args.argos.exists():
        raise SystemExit(f"{args.argos} bulunamadı")

    schema = json.loads(args.schema.read_text(encoding="utf-8"))
    Draft202012Validator.check_schema(schema)
    validator = Draft202012Validator(schema, format_checker=FormatChecker())

    val_rows = load_jsonl(args.data)
    jobs = []
    for i, row in enumerate(val_rows):
        if "metin" not in row or "cikti" not in row:
            raise SystemExit(f"{args.data}:{i + 1} metin/cikti eksik")
        jobs.append((i, row["cikti"], row["metin"]))

    print(f"val.jsonl: {len(jobs)} örnek, eşzamanlı={args.concurrency}", file=sys.stderr)
    val_records = run_pool(
        jobs,
        args.endpoint,
        args.token,
        args.api,
        args.max_tokens,
        args.timeout,
        validator,
        args.concurrency,
    )
    val_summary = summarize(val_records)

    argos_gold = json.loads(args.argos.read_text(encoding="utf-8"))
    print("MEGEP Argos örneği (eğitim kümesinde yok)", file=sys.stderr)
    argos_records = run_pool(
        [(0, argos_gold, argos_source_text(argos_gold))],
        args.endpoint,
        args.token,
        args.api,
        args.max_tokens,
        args.timeout,
        validator,
        1,
    )
    argos_summary = summarize(argos_records)

    print()
    print_table(
        [
            (args.data.name, val_summary),
            ("ornek-argos", argos_summary),
        ]
    )

    stamp = datetime.now(timezone.utc).strftime("%Y-%m-%dT%H%M%SZ")
    args.out_dir.mkdir(parents=True, exist_ok=True)
    out_path = args.out_dir / f"eval_{stamp}.json"
    report = {
        "tarih": stamp,
        "endpoint": args.endpoint,
        "api": args.api,
        "data": str(args.data),
        "argos": str(args.argos),
        "concurrency": args.concurrency,
        "val": {
            **val_summary,
            "ornekler": [
                {
                    "index": r["index"],
                    "valid_json": r["valid_json"],
                    "schema_ok": r["schema_ok"],
                    "field_match": r["field_match"],
                    "error": r.get("error"),
                }
                for r in val_records
            ],
        },
        "ornek_argos": {
            **argos_summary,
            "ornekler": [
                {
                    "index": r["index"],
                    "valid_json": r["valid_json"],
                    "schema_ok": r["schema_ok"],
                    "field_match": r["field_match"],
                    "error": r.get("error"),
                }
                for r in argos_records
            ],
        },
    }
    out_path.write_text(json.dumps(report, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print(f"\nyazıldı {out_path}")


if __name__ == "__main__":
    main()
