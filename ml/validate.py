"""Üretilen jsonl çıktılarını kontrat.json şemasına karşı doğrular ve istatistik basar."""

from __future__ import annotations

import argparse
import json
import sys
from collections import Counter
from pathlib import Path

from jsonschema import Draft202012Validator, FormatChecker

KOK = Path(__file__).resolve().parent
SEMA_YOL = KOK / "schema" / "kontrat.json"
ORNEK_YOL = KOK / "schema" / "ornek-argos.json"


def oku_jsonl(yol: Path) -> list[dict]:
    satirlar = []
    with yol.open(encoding="utf-8") as f:
        for i, line in enumerate(f, 1):
            line = line.strip()
            if not line:
                continue
            try:
                satirlar.append(json.loads(line))
            except json.JSONDecodeError as e:
                raise SystemExit(f"{yol}:{i} JSON okunamadı: {e}") from e
    return satirlar


def yol_var(obj: dict, path: str) -> bool:
    cur: object = obj
    for parca in path.split("."):
        if not isinstance(cur, dict) or parca not in cur:
            return False
        cur = cur[parca]
    if cur is None:
        return False
    if cur == "" or cur == []:
        return False
    return True


def gurultu_etiket(g: str) -> str:
    if g.startswith("eksik_alan"):
        return "eksik_alan"
    if g.startswith("tarih_celiskisi"):
        return "tarih_celiskisi"
    return g


def istatistik(ornekler: list[dict], etiket: str) -> None:
    n = len(ornekler)
    if n == 0:
        print(f"{etiket}: boş")
        return
    diller = Counter(o["meta"]["dil"] for o in ornekler)
    sablon = Counter(o["meta"]["sablon_no"] for o in ornekler)
    gurultu = Counter()
    for o in ornekler:
        etiketler = o["meta"].get("eklenen_gurultu") or []
        if not etiketler:
            gurultu["(yok)"] += 1
        for g in etiketler:
            gurultu[gurultu_etiket(g)] += 1

    alanlar = [
        "meta",
        "meta.otel_adi",
        "donem.baslangic",
        "donem.bitis",
        "donem.alt_donemler",
        "oda_kontenjanlari",
        "fiyatlar",
        "release.gun",
        "release.kapsam",
        "stop_sale",
    ]
    print(f"\n=== {etiket} (n={n}) ===")
    print("dil:")
    for k, v in sorted(diller.items()):
        print(f"  {k}: {v} ({100.0 * v / n:.1f}%)")
    print("şablon:")
    for k, v in sorted(sablon.items()):
        print(f"  {k}: {v} ({100.0 * v / n:.1f}%)")
    print("gürültü (bir örnekte birden fazla olabilir):")
    for k, v in sorted(gurultu.items()):
        print(f"  {k}: {v} ({100.0 * v / n:.1f}%)")
    print("alan doluluk:")
    for path in alanlar:
        dolu = sum(1 for o in ornekler if yol_var(o["cikti"], path))
        print(f"  {path}: {dolu}/{n} ({100.0 * dolu / n:.1f}%)")


def dogrula(ornekler: list[dict], validator: Draft202012Validator, kaynak: Path) -> int:
    hata = 0
    for i, o in enumerate(ornekler, 1):
        if not isinstance(o, dict) or "cikti" not in o or "metin" not in o or "meta" not in o:
            print(f"{kaynak}:{i} satırda metin/cikti/meta eksik", file=sys.stderr)
            hata += 1
            continue
        errs = sorted(validator.iter_errors(o["cikti"]), key=lambda e: list(e.path))
        for e in errs:
            yol = ".".join(str(p) for p in e.path) or "(kök)"
            print(f"{kaynak}:{i} {yol}: {e.message}", file=sys.stderr)
            hata += 1
    return hata


def main() -> None:
    p = argparse.ArgumentParser(description="Sentetik çıktıları şemaya karşı doğrular.")
    p.add_argument("--data-dir", type=Path, default=KOK / "data")
    p.add_argument("--schema", type=Path, default=SEMA_YOL)
    args = p.parse_args()

    sema = json.loads(args.schema.read_text(encoding="utf-8"))
    Draft202012Validator.check_schema(sema)
    validator = Draft202012Validator(sema, format_checker=FormatChecker())

    argos = json.loads(ORNEK_YOL.read_text(encoding="utf-8"))
    argos_hata = sorted(validator.iter_errors(argos), key=lambda e: list(e.path))
    if argos_hata:
        for e in argos_hata:
            print(f"ornek-argos.json {e.message}", file=sys.stderr)
        raise SystemExit("MEGEP örneği şemaya uymuyor")
    print("ornek-argos.json şemaya uyuyor (eğitim kümesine dahil değil)")

    hata = 0
    kume = []
    for ad in ("train.jsonl", "val.jsonl"):
        yol = args.data_dir / ad
        if not yol.exists():
            print(f"{yol} yok; önce python generate.py çalıştırın", file=sys.stderr)
            raise SystemExit(1)
        ornekler = oku_jsonl(yol)
        hata += dogrula(ornekler, validator, yol)
        istatistik(ornekler, ad)
        kume.extend(ornekler)

    istatistik(kume, "toplam")
    if hata:
        print(f"\n{hata} şema hatası", file=sys.stderr)
        raise SystemExit(1)
    print("\ntüm cikti değerleri şemaya uyuyor")


if __name__ == "__main__":
    main()
