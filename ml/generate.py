"""Şemadan rastgele kontenjan sözleşmesi nesnesi ve şablon metni üretir.

LLM kullanılmaz; çıktı --seed ile deterministiktir.
MEGEP Argos örneği bu hatta karıştırılmaz.
"""

from __future__ import annotations

import argparse
import json
import random
from datetime import date, timedelta
from pathlib import Path

from faker import Faker

KOK = Path(__file__).resolve().parent
ODA_TIPLERI = ["standart", "suit", "aile", "deluxe", "balayi", "engelli"]
PANSIYONLAR = ["RO", "BB", "HB", "FB", "AI"]
UYUMSUZ_ODA = ["tek_kisilik", "iki_kisilik", "uc_kisilik"]
SOZLESME_TIPLERI = [
    "tamamen_garantili",
    "kismen_garantili",
    "garantisiz",
    "istege_bagli",
    "serbest_satis",
    "blok_rezervasyon",
    "blok_satin_alma",
    "belirtilmemis",
]
PARA = ["EUR", "GBP", "USD", "TRY"]
KUR = ["giris_gunu_tcmb", "cikis_gunu_tcmb", "sabit_kur", "belirtilmemis"]
RELEASE_KAPSAM = ["isim_listesi", "kontenjan_iadesi", "her_ikisi", "belirtilmemis"]
BILDIRIM = ["yazili", "faks", "eposta", "sistem", "belirtilmemis"]
ALT_AD_TR = ["Erken sezon", "Orta sezon", "Yüksek sezon", "Geç sezon"]
ALT_AD_EN = ["Early season", "Shoulder season", "High season", "Late season"]
AY_TR = {
    1: "Ocak",
    2: "Şubat",
    3: "Mart",
    4: "Nisan",
    5: "Mayıs",
    6: "Haziran",
    7: "Temmuz",
    8: "Ağustos",
    9: "Eylül",
    10: "Ekim",
    11: "Kasım",
    12: "Aralık",
}
AY_EN = {
    1: "January",
    2: "February",
    3: "March",
    4: "April",
    5: "May",
    6: "June",
    7: "July",
    8: "August",
    9: "September",
    10: "October",
    11: "November",
    12: "December",
}
PANSIYON_AD_TR = {"RO": "sadece oda", "BB": "oda kahvaltı", "HB": "yarım pansiyon", "FB": "tam pansiyon", "AI": "her şey dahil"}
PANSIYON_AD_EN = {
    "RO": "room only",
    "BB": "bed and breakfast",
    "HB": "half board",
    "FB": "full board",
    "AI": "all inclusive",
}
BIRIM_TR = {"oda_gecelik": "oda / gece", "kisi_gecelik": "kişi / gece"}
BIRIM_EN = {"oda_gecelik": "per room per night", "kisi_gecelik": "per person per night"}
ODA_TIPI_EN = {
    "standart": "standard",
    "suit": "suite",
    "aile": "family",
    "deluxe": "penthouse",
    "balayi": "honeymoon",
    "engelli": "accessible",
    "tek_kisilik": "single",
    "iki_kisilik": "double",
    "uc_kisilik": "triple",
}
KAPSAM_EN = {
    "tüm tesis": "entire property",
    "standart": "standard",
    "suit": "suite",
}
SEHIR_TR = ["Antalya", "Muğla", "İzmir", "İstanbul", "Ankara", "Aydın", "Mersin"]
SEHIR_EN = ["Antalya", "London", "Manchester", "Amsterdam", "Berlin"]


def iso(d: date) -> str:
    return d.isoformat()


def parse_iso(s: str) -> date:
    y, m, d = (int(p) for p in s.split("-"))
    return date(y, m, d)


def fmt_tarih(s: str | None, bicim: str, dil: str) -> str:
    if not s:
        return ""
    d = parse_iso(s)
    if bicim == "iso":
        return s
    if bicim == "nokta":
        return f"{d.day:02d}.{d.month:02d}.{d.year}"
    if dil == "tr":
        return f"{d.day} {AY_TR[d.month]} {d.year}"
    return f"{d.day} {AY_EN[d.month]} {d.year}"


def fmt_sayi(n: int | float, bicim: str) -> str:
    if isinstance(n, float) and not n.is_integer():
        if bicim == "en":
            return f"{n:,.2f}"
        if bicim == "tr":
            tam, kesir = f"{n:.2f}".split(".")
            tam_fmt = fmt_sayi(int(tam), "tr")
            return f"{tam_fmt},{kesir}"
        return f"{n:.2f}"
    i = int(n)
    if bicim == "tr" and i >= 1000:
        return f"{i:,}".replace(",", ".")
    if bicim == "en" and i >= 1000:
        return f"{i:,}"
    return str(i)


def metin_oda_tipi(ad: str, dil: str, adet: int | None = None) -> str:
    if dil != "en":
        return ad
    if ad == "suit" and adet is not None and int(adet) % 2 == 1:
        return "junior suite"
    return ODA_TIPI_EN.get(ad, ad)


def kontenjan_adet(obj: dict, tip: str) -> int | None:
    for k in obj.get("oda_kontenjanlari") or []:
        if k.get("oda_tipi") == tip:
            return k.get("adet")
    return None


def metin_kapsam(ad: str, dil: str) -> str:
    if dil == "en":
        return KAPSAM_EN.get(ad, ad)
    return ad


def fmt_para(tutar: str, para: str, yapisik: bool) -> str:
    if not para:
        return tutar
    if yapisik:
        return f"{tutar}{para}"
    return f"{tutar} {para}"


def yaz_aralik(rng: random.Random) -> tuple[date, date, str]:
    yil = rng.randint(2024, 2027)
    bas = date(yil, 4, rng.choice([1, 5, 10, 15]))
    bit = date(yil, 10, rng.choice([15, 20, 25, 31]))
    return bas, bit, "yaz"


def kis_aralik(rng: random.Random) -> tuple[date, date, str]:
    yil = rng.randint(2024, 2027)
    bas = date(yil, 11, rng.choice([1, 10, 15]))
    bit_yil = yil + 1
    ay = rng.choice([1, 2, 3])
    gun = 28 if ay == 2 else rng.choice([15, 20, 28])
    bit = date(bit_yil, ay, gun)
    return bas, bit, "kis"


def alt_donem_kes(bas: date, bit: date, adlar: list[str], rng: random.Random) -> list[dict]:
    n = rng.randint(2, min(4, len(adlar)))
    toplam = (bit - bas).days
    if toplam < n * 10:
        return []
    kesikler = sorted(rng.sample(range(8, toplam - 8), n - 1))
    noktalar = [bas] + [bas + timedelta(days=k) for k in kesikler] + [bit + timedelta(days=1)]
    out = []
    for i in range(n):
        a = noktalar[i]
        b = noktalar[i + 1] - timedelta(days=1)
        if b <= a:
            return []
        out.append({"ad": adlar[i], "baslangic": iso(a), "bitis": iso(b)})
    return out


def stop_sale_araliklari(bas: date, bit: date, rng: random.Random) -> list[dict]:
    if rng.random() >= 0.5:
        return []
    n = rng.randint(0, 3)
    out = []
    span = (bit - bas).days
    if span < 14 or n == 0:
        return []
    for _ in range(n):
        offset = rng.randint(0, max(1, span - 7))
        uzun = rng.randint(2, min(14, span - offset))
        a = bas + timedelta(days=offset)
        b = a + timedelta(days=uzun)
        if b > bit:
            b = bit
        if b <= a:
            continue
        out.append(
            {
                "baslangic": iso(a),
                "bitis": iso(b),
                "kapsam": rng.choice(["tüm tesis", "standart", "suit"]),
                "bildirim_yontemi": rng.choice(BILDIRIM),
            }
        )
    return out


def isimler(fake: Faker, dil: str, rng: random.Random) -> tuple[str, str, str]:
    soy = fake.last_name()
    otel = f"{soy} Otel" if dil == "tr" else f"{soy} Hotel"
    if dil == "tr":
        acente = f"{fake.last_name()} Turizm"
        sehir = rng.choice(SEHIR_TR)
    else:
        acente = f"{fake.last_name()} Travel"
        sehir = rng.choice(SEHIR_EN)
    return otel, acente, sehir


def asama_a(rng: random.Random, fake: Faker, dil: str) -> dict:
    otel, acente, sehir = isimler(fake, dil, rng)
    if rng.random() < 0.5:
        bas, bit, sezon = yaz_aralik(rng)
    else:
        bas, bit, sezon = kis_aralik(rng)
    adlar = list(ALT_AD_TR if dil == "tr" else ALT_AD_EN)
    alt = []
    if rng.random() < 0.4:
        alt = alt_donem_kes(bas, bit, adlar, rng)

    tipler = rng.sample(ODA_TIPLERI, k=rng.randint(2, 5))
    kontenjan = []
    for t in tipler:
        item = {"oda_tipi": t, "adet": rng.randint(1, 300)}
        if t == "engelli" and rng.random() < 0.5:
            item["aciklama"] = "Sözleşmede 'özürlü odası' olarak geçer" if dil == "tr" else "Listed as accessible room"
        if t == "standart" and rng.random() < 0.3:
            item["aciklama"] = "Sözleşmede 'normal oda' olarak geçer" if dil == "tr" else "Listed as standard room"
        kontenjan.append(item)

    birim = rng.choice(["oda_gecelik", "kisi_gecelik"])
    fiyatlar = []
    for t in tipler:
        pansiyonlar = rng.sample(PANSIYONLAR, k=rng.randint(1, 3))
        for p in pansiyonlar:
            taban = rng.randint(45, 280)
            if t in ("suit", "deluxe", "balayi"):
                taban += rng.randint(40, 220)
            if alt:
                for a in alt:
                    tutar = float(taban + rng.randint(-15, 40))
                    if rng.random() < 0.3:
                        tutar += 0.5
                    fiyatlar.append(
                        {
                            "oda_tipi": t,
                            "pansiyon": p,
                            "tutar": tutar if tutar > 0 else float(taban),
                            "birim": birim,
                            "alt_donem_ad": a["ad"],
                        }
                    )
            else:
                tutar = float(taban)
                if rng.random() < 0.25:
                    tutar += 0.5
                fiyatlar.append({"oda_tipi": t, "pansiyon": p, "tutar": tutar, "birim": birim})

    gun = rng.randint(3, 30)
    kapsam = rng.choice(RELEASE_KAPSAM)
    if dil == "tr":
        kaynak = (
            f"Acente, oda kontenjanı ile ilgili son durumu müşterilerin otele girişini "
            f"{gun} gün önce belirler ve isim listesi ile birlikte otele bildirir."
        )
    else:
        kaynak = (
            f"The release period is {gun} days prior to arrival. Name lists covering "
            f"the allotment must reach the Hotel before that deadline."
        )

    donem = {"baslangic": iso(bas), "bitis": iso(bit)}
    if alt:
        donem["alt_donemler"] = alt

    imza = bas - timedelta(days=rng.randint(10, 90))
    meta = {
        "otel_adi": otel,
        "acente_adi": acente,
        "sozlesme_tipi": rng.choice(SOZLESME_TIPLERI),
        "sezon": sezon,
        "para_birimi": rng.choice(PARA),
        "kur_esasi": rng.choice(KUR),
        "yetkili_mahkeme": sehir,
        "imza_tarihi": iso(imza) if rng.random() < 0.6 else None,
    }

    return {
        "meta": meta,
        "donem": donem,
        "oda_kontenjanlari": kontenjan,
        "fiyatlar": fiyatlar,
        "release": {"gun": gun, "kapsam": kapsam, "kaynak_ifade": kaynak},
        "stop_sale": stop_sale_araliklari(bas, bit, rng),
    }


def gurultu_eksik(obj: dict, rng: random.Random) -> str:
    """Şemayı bozmadan bir alanı boşaltır; metin de bunu yansıtır."""
    secim = rng.choice(
        ["donem_tarih", "meta", "stop_sale", "release_kapsam", "alt_donem", "pansiyon"]
    )
    if secim == "donem_tarih":
        obj["donem"]["baslangic"] = None
        obj["donem"]["bitis"] = None
        obj["donem"].pop("alt_donemler", None)
        for f in obj["fiyatlar"]:
            f.pop("alt_donem_ad", None)
        return "eksik_alan:donem"
    if secim == "meta":
        obj.pop("meta", None)
        return "eksik_alan:meta"
    if secim == "stop_sale":
        obj["stop_sale"] = []
        return "eksik_alan:stop_sale"
    if secim == "release_kapsam":
        obj["release"].pop("kapsam", None)
        obj["release"].pop("kaynak_ifade", None)
        return "eksik_alan:release.kapsam"
    if secim == "alt_donem":
        if not obj["donem"].get("alt_donemler"):
            obj["donem"]["baslangic"] = None
            obj["donem"]["bitis"] = None
            return "eksik_alan:donem"
        obj["donem"].pop("alt_donemler", None)
        for f in obj["fiyatlar"]:
            f.pop("alt_donem_ad", None)
        return "eksik_alan:donem.alt_donemler"
    for f in obj["fiyatlar"]:
        f.pop("pansiyon", None)
    return "eksik_alan:fiyatlar.pansiyon"


def gurultu_tarih(obj: dict, rng: random.Random) -> str | None:
    d = obj["donem"]
    if not d.get("baslangic") or not d.get("bitis"):
        return None
    alt = d.get("alt_donemler") or []
    if alt and rng.random() < 0.5:
        hedef = rng.choice(alt)
        bas = parse_iso(d["baslangic"]) - timedelta(days=rng.randint(10, 25))
        hedef["baslangic"] = iso(bas)
        return "tarih_celiskisi:alt_donem_disarida"
    # bitiş < başlangıç
    d["baslangic"], d["bitis"] = d["bitis"], d["baslangic"]
    return "tarih_celiskisi:bitis_baslangictan_once"


def gurultu_oda_uyumsuz(obj: dict, rng: random.Random) -> None:
    birim = obj["fiyatlar"][0]["birim"] if obj["fiyatlar"] else "oda_gecelik"
    extra = {
        "oda_tipi": rng.choice(UYUMSUZ_ODA),
        "tutar": float(rng.randint(50, 120)),
        "birim": birim,
    }
    if obj["fiyatlar"] and "pansiyon" in obj["fiyatlar"][0]:
        extra["pansiyon"] = rng.choice(PANSIYONLAR + ["belirtilmemis"])
    obj["fiyatlar"].append(extra)


def fazladan_maddeler(obj: dict, dil: str, sayi_b: str) -> str:
    meta = obj.get("meta") or {}
    sehir = meta.get("yetkili_mahkeme") or ("Antalya" if dil == "tr" else "Antalya")
    gun = 15 if sum(ord(c) for c in sehir) % 2 == 0 else 30
    if dil == "tr":
        return (
            f"\nÖDEME: Faturalar düzenlenme tarihinden itibaren {fmt_sayi(gun, sayi_b)} gün içinde "
            f"ödenir. Avans bu sözleşmede öngörülmemiştir.\n"
            f"TEBLİGAT: Taraflar tebligatlarını yazılı olarak veya e-posta yoluyla yapar.\n"
            f"YETKİLİ MAHKEME: İşbu sözleşmeden doğacak uyuşmazlıklarda {sehir} Mahkemeleri ve "
            f"İcra Daireleri yetkilidir.\n"
        )
    return (
        f"\nPAYMENT: Invoices are payable within {fmt_sayi(gun, sayi_b)} days of issue. "
        f"No advance payment is required under this contract.\n"
        f"NOTICES: Notices shall be given in writing or by e-mail.\n"
        f"JURISDICTION: The courts of {sehir} shall have exclusive jurisdiction.\n"
    )


# --- metin bölümleri ---


def otel_acente(obj: dict, dil: str) -> tuple[str, str]:
    meta = obj.get("meta") or {}
    if dil == "tr":
        return meta.get("otel_adi") or "Otel", meta.get("acente_adi") or "Acente"
    return meta.get("otel_adi") or "the Hotel", meta.get("acente_adi") or "the Agent"


def donem_cumle(obj: dict, dil: str, tarih_b: str) -> str | None:
    d = obj["donem"]
    if not d.get("baslangic") or not d.get("bitis"):
        return None
    a = fmt_tarih(d["baslangic"], tarih_b, dil)
    b = fmt_tarih(d["bitis"], tarih_b, dil)
    if dil == "tr":
        return f"İşbu sözleşme {a} tarihinde başlar ve {b} tarihinde sona erer."
    return f"This contract runs from {a} to {b}."


def alt_donem_metin(obj: dict, dil: str, tarih_b: str) -> str:
    alt = obj["donem"].get("alt_donemler") or []
    if not alt:
        return ""
    satirlar = []
    for a in alt:
        aralik = f"{fmt_tarih(a['baslangic'], tarih_b, dil)} – {fmt_tarih(a['bitis'], tarih_b, dil)}"
        satirlar.append(f"{a['ad']}: {aralik}")
    if dil == "tr":
        return "Sezon içi fiyat dönemleri:\n" + "\n".join(satirlar)
    return "Seasonal rate periods:\n" + "\n".join(satirlar)


def kontenjan_tablo(obj: dict, dil: str, sayi_b: str) -> str:
    baslik = "Oda tipi | Adet" if dil == "tr" else "Room type | Allotment"
    cizgi = "---------|-----" if dil == "tr" else "----------|----------"
    satir = [baslik, cizgi]
    for k in obj["oda_kontenjanlari"]:
        ad = metin_oda_tipi(k["oda_tipi"], dil, k.get("adet"))
        if k.get("aciklama"):
            ad = f"{ad} ({k['aciklama']})"
        satir.append(f"{ad} | {fmt_sayi(k['adet'], sayi_b)}")
    return "\n".join(satir)


def kontenjan_madde(obj: dict, dil: str, sayi_b: str) -> str:
    parca = []
    for k in obj["oda_kontenjanlari"]:
        extra = f" ({k['aciklama']})" if k.get("aciklama") else ""
        parca.append(f"{fmt_sayi(k['adet'], sayi_b)} {metin_oda_tipi(k['oda_tipi'], dil, k.get('adet'))}{extra}")
    liste = ", ".join(parca)
    if dil == "tr":
        return f"Otel, acenteye şu oda kontenjanını tahsis eder: {liste}."
    return f"The Hotel allocates the following allotment to the Agent: {liste}."


def fiyat_tablo(obj: dict, dil: str, sayi_b: str, yapisik: bool) -> str:
    para = (obj.get("meta") or {}).get("para_birimi") or ""
    if dil == "tr":
        baslik = "Oda tipi | Pansiyon | Tutar | Birim | Dönem"
        cizgi = "---------|----------|-------|-------|------"
    else:
        baslik = "Room type | Board | Rate | Unit | Period"
        cizgi = "----------|-------|------|------|--------"
    satir = [baslik, cizgi]
    for f in obj["fiyatlar"]:
        p = f.get("pansiyon") or ("belirtilmemiş" if dil == "tr" else "unspecified")
        birim = BIRIM_TR.get(f["birim"], f["birim"]) if dil == "tr" else BIRIM_EN.get(f["birim"], f["birim"])
        donem = f.get("alt_donem_ad") or "-"
        tutar = fmt_para(fmt_sayi(f["tutar"], sayi_b), para, yapisik)
        satir.append(
            f"{metin_oda_tipi(f['oda_tipi'], dil, kontenjan_adet(obj, f['oda_tipi']))} | {p} | {tutar} | {birim} | {donem}"
        )
    return "\n".join(satir)


def fiyat_paragraf(obj: dict, dil: str, sayi_b: str, yapisik: bool) -> str:
    para = (obj.get("meta") or {}).get("para_birimi") or ""
    cumleler = []
    for f in obj["fiyatlar"]:
        tutar = fmt_para(fmt_sayi(f["tutar"], sayi_b), para, yapisik)
        birim = BIRIM_TR.get(f["birim"], f["birim"]) if dil == "tr" else BIRIM_EN.get(f["birim"], f["birim"])
        p = f.get("pansiyon")
        oda = metin_oda_tipi(f["oda_tipi"], dil, kontenjan_adet(obj, f["oda_tipi"]))
        if dil == "tr":
            pan = f", {PANSIYON_AD_TR.get(p, p)}" if p and p != "belirtilmemis" else ""
            donem = f" ({f['alt_donem_ad']})" if f.get("alt_donem_ad") else ""
            cumleler.append(f"{oda}{pan}{donem} için {tutar} {birim}")
        else:
            pan = f", {PANSIYON_AD_EN.get(p, p)}" if p and p != "belirtilmemis" else ""
            donem = f" ({f['alt_donem_ad']})" if f.get("alt_donem_ad") else ""
            cumleler.append(f"{oda}{pan}{donem} at {tutar} {birim}")
    if dil == "tr":
        return "Fiyatlar: " + "; ".join(cumleler) + "."
    unit = BIRIM_EN.get(obj["fiyatlar"][0]["birim"], "per room per night") if obj.get("fiyatlar") else "per room per night"
    return f"Rates are {unit}. " + "; ".join(cumleler) + "."


def release_cumle(obj: dict, dil: str) -> str:
    r = obj["release"]
    if r.get("kaynak_ifade"):
        return r["kaynak_ifade"]
    gun = r["gun"]
    if dil == "tr":
        return (
            f"Acente, oda kontenjanı ile ilgili son durumu müşterilerin otele girişini "
            f"{gun} gün önce belirler ve isim listesi ile birlikte otele bildirir."
        )
    return (
        f"The release period is {gun} days prior to arrival. Unsold allotment reverts "
        f"to the Hotel unless covered by a name list."
    )


def stop_sale_metin(obj: dict, dil: str, tarih_b: str) -> str:
    ss = obj.get("stop_sale") or []
    if not ss:
        return ""
    satirlar = []
    for s in ss:
        a = fmt_tarih(s.get("baslangic"), tarih_b, dil)
        b = fmt_tarih(s.get("bitis"), tarih_b, dil)
        kapsam = metin_kapsam(s.get("kapsam") or "", dil)
        satirlar.append(f"{a} – {b}" + (f" ({kapsam})" if kapsam else ""))
    if dil == "tr":
        return (
            "Otel, aşağıdaki tarih aralıklarında stop-sale uygulayabilir; bildirim yazılı "
            "veya e-posta ile yapılır:\n" + "\n".join(f"- {x}" for x in satirlar)
        )
    return (
        "Stop-sale periods (the Hotel may close sales on the dates below; notice in "
        "writing or by e-mail):\n" + "\n".join(f"- {x}" for x in satirlar)
    )


def baslik(obj: dict, dil: str) -> str:
    otel, acente = otel_acente(obj, dil)
    if dil == "tr":
        return f"KONTENJAN SÖZLEŞMESİ\n{acente} – {otel}\n"
    return f"ALLOTMENT CONTRACT\n{acente} – {otel}\n"


def sablon_maddeli(obj: dict, dil: str, tarih_b: str, sayi_b: str, yapisik: bool) -> str:
    """MEGEP örnek sözleşmesine yakın madde madde dil."""
    otel, acente = otel_acente(obj, dil)
    parcalar = [baslik(obj, dil)]
    n = 1
    donem = donem_cumle(obj, dil, tarih_b)
    if dil == "tr":
        parcalar.append(
            f"İşbu sözleşme, {acente} ile {otel} arasında aşağıda belirtilen koşullarla akdedilmiştir.\n"
        )
        if donem:
            parcalar.append(f"MADDE {n} — SÖZLEŞME SÜRESİ\n{donem}")
            alt = alt_donem_metin(obj, dil, tarih_b)
            if alt:
                parcalar.append(alt)
            n += 1
        parcalar.append(f"MADDE {n} — ODA KONTENJANI\n{kontenjan_madde(obj, dil, sayi_b)}")
        n += 1
        parcalar.append(f"MADDE {n} — İSİM LİSTESİ\n{release_cumle(obj, dil)}")
        n += 1
        ss = stop_sale_metin(obj, dil, tarih_b)
        if ss:
            parcalar.append(f"MADDE {n} — STOP-SALE\n{ss}")
            n += 1
        parcalar.append(f"MADDE {n} — FİYATLAR\n{fiyat_paragraf(obj, dil, sayi_b, yapisik)}")
    else:
        parcalar.append(f"This contract is entered into by {acente} and {otel} on the terms below.\n")
        if donem:
            parcalar.append(f"ARTICLE {n} — TERM\n{donem}")
            alt = alt_donem_metin(obj, dil, tarih_b)
            if alt:
                parcalar.append(alt)
            n += 1
        parcalar.append(f"ARTICLE {n} — ALLOTMENT\n{kontenjan_madde(obj, dil, sayi_b)}")
        n += 1
        parcalar.append(f"ARTICLE {n} — NAME LIST\n{release_cumle(obj, dil)}")
        n += 1
        ss = stop_sale_metin(obj, dil, tarih_b)
        if ss:
            parcalar.append(f"ARTICLE {n} — STOP-SALE\n{ss}")
            n += 1
        parcalar.append(f"ARTICLE {n} — RATES\n{fiyat_paragraf(obj, dil, sayi_b, yapisik)}")
    return "\n\n".join(parcalar) + "\n"


def sablon_tablo(obj: dict, dil: str, tarih_b: str, sayi_b: str, yapisik: bool) -> str:
    parcalar = [baslik(obj, dil)]
    donem = donem_cumle(obj, dil, tarih_b)
    if donem:
        parcalar.append(donem)
        alt = alt_donem_metin(obj, dil, tarih_b)
        if alt:
            parcalar.append(alt)
    if dil == "tr":
        parcalar.append("Kontenjan tablosu\n" + kontenjan_tablo(obj, dil, sayi_b))
        parcalar.append("Fiyat tablosu\n" + fiyat_tablo(obj, dil, sayi_b, yapisik))
    else:
        parcalar.append("Allotment table\n" + kontenjan_tablo(obj, dil, sayi_b))
        parcalar.append("Rate table\n" + fiyat_tablo(obj, dil, sayi_b, yapisik))
    parcalar.append(release_cumle(obj, dil))
    ss = stop_sale_metin(obj, dil, tarih_b)
    if ss:
        parcalar.append(ss)
    return "\n\n".join(parcalar) + "\n"


def sablon_paragraf(obj: dict, dil: str, tarih_b: str, sayi_b: str, yapisik: bool) -> str:
    otel, acente = otel_acente(obj, dil)
    donem = donem_cumle(obj, dil, tarih_b)
    if dil == "tr":
        giris = (
            f"{acente}, {otel} işletmesi ile kontenjan esasına göre çalışmayı kararlaştırmıştır. "
        )
        if donem:
            giris += donem + " "
        alt = alt_donem_metin(obj, dil, tarih_b)
        govde = giris + kontenjan_madde(obj, dil, sayi_b) + " " + release_cumle(obj, dil)
        if alt:
            govde += "\n\n" + alt
        govde += "\n\n" + fiyat_paragraf(obj, dil, sayi_b, yapisik)
    else:
        giris = f"{acente} and {otel} agree to cooperate on an allotment basis. "
        if donem:
            giris += donem + " "
        alt = alt_donem_metin(obj, dil, tarih_b)
        govde = giris + kontenjan_madde(obj, dil, sayi_b) + " " + release_cumle(obj, dil)
        if alt:
            govde += "\n\n" + alt
        govde += "\n\n" + fiyat_paragraf(obj, dil, sayi_b, yapisik)
    ss = stop_sale_metin(obj, dil, tarih_b)
    if ss:
        govde += "\n\n" + ss
    return govde + "\n"


def sablon_karisik(obj: dict, dil: str, tarih_b: str, sayi_b: str, yapisik: bool) -> str:
    """Üstte kısa maddeler, altta tablo."""
    ust = sablon_maddeli(obj, dil, tarih_b, sayi_b, yapisik)
    if dil == "tr":
        return ust + "\nEK — FİYAT VE KONTENJAN TABLOLARI\n\n" + kontenjan_tablo(obj, dil, sayi_b) + "\n\n" + fiyat_tablo(obj, dil, sayi_b, yapisik) + "\n"
    return ust + "\nANNEX — RATE AND ALLOTMENT TABLES\n\n" + kontenjan_tablo(obj, dil, sayi_b) + "\n\n" + fiyat_tablo(obj, dil, sayi_b, yapisik) + "\n"


def sablon_ekler(obj: dict, dil: str, tarih_b: str, sayi_b: str, yapisik: bool) -> str:
    otel, acente = otel_acente(obj, dil)
    donem = donem_cumle(obj, dil, tarih_b) or ""
    if dil == "tr":
        return (
            f"{acente} / {otel}\nKontenjan sözleşmesi ekleri\n{donem}\n\n"
            f"EK-1 KONTENJAN\n{kontenjan_tablo(obj, dil, sayi_b)}\n\n"
            f"EK-2 FİYATLAR\n{fiyat_tablo(obj, dil, sayi_b, yapisik)}\n\n"
            f"EK-3 RELEASE\n{release_cumle(obj, dil)}\n\n"
            + ((stop_sale_metin(obj, dil, tarih_b) + "\n") if obj.get("stop_sale") else "")
        )
    return (
        f"{acente} / {otel}\nAllotment contract annexes\n{donem}\n\n"
        f"ANNEX-1 ALLOTMENT\n{kontenjan_tablo(obj, dil, sayi_b)}\n\n"
        f"ANNEX-2 RATES\n{fiyat_tablo(obj, dil, sayi_b, yapisik)}\n\n"
        f"ANNEX-3 RELEASE\n{release_cumle(obj, dil)}\n\n"
        + ((stop_sale_metin(obj, dil, tarih_b) + "\n") if obj.get("stop_sale") else "")
    )


def sablon_mektup(obj: dict, dil: str, tarih_b: str, sayi_b: str, yapisik: bool) -> str:
    otel, acente = otel_acente(obj, dil)
    donem = donem_cumle(obj, dil, tarih_b)
    if dil == "tr":
        giris = (
            f"Sayın yetkili,\n\n{otel} olarak {acente} ile yapılacak kontenjan çalışmasının "
            f"koşulları aşağıdadır. "
        )
        if donem:
            giris += donem + " "
        giris += (
            "Acente, tahsis edilen odaları kendi müşterisine pazarlar; otel, kapasite dahilinde "
            "oda bulundurmayı taahhüt eder.\n\n"
        )
        return (
            giris
            + kontenjan_madde(obj, dil, sayi_b)
            + "\n\n"
            + fiyat_tablo(obj, dil, sayi_b, yapisik)
            + "\n\n"
            + release_cumle(obj, dil)
            + "\n\n"
            + (stop_sale_metin(obj, dil, tarih_b) + "\n" if obj.get("stop_sale") else "")
            + "\nSaygılarımızla,\n"
        )
    giris = (
        f"Dear Sir/Madam,\n\nPlease find below the allotment terms agreed between {otel} "
        f"and {acente}. "
    )
    if donem:
        giris += donem + " "
    giris += "The Agent markets the allocated rooms; the Hotel undertakes to hold the inventory.\n\n"
    return (
        giris
        + kontenjan_madde(obj, dil, sayi_b)
        + "\n\n"
        + fiyat_tablo(obj, dil, sayi_b, yapisik)
        + "\n\n"
        + release_cumle(obj, dil)
        + "\n\n"
        + (stop_sale_metin(obj, dil, tarih_b) + "\n" if obj.get("stop_sale") else "")
        + "\nYours faithfully,\n"
    )


def _en_rate_unit(obj: dict) -> str:
    if obj.get("fiyatlar"):
        return BIRIM_EN.get(obj["fiyatlar"][0]["birim"], "per room per night")
    return "per room per night"


def sablon_en_articles(obj: dict, dil: str, tarih_b: str, sayi_b: str, yapisik: bool) -> str:
    """Tour-operator allotment agreement: ARTICLE N / The Hotel allocates."""
    otel, acente = otel_acente(obj, "en")
    para = (obj.get("meta") or {}).get("para_birimi") or "EUR"
    n = 1
    parcalar = [
        f"ALLOTMENT AGREEMENT\nbetween {acente} (the Agent) and {otel} (the Hotel)\n",
        "The Parties agree the allotment of rooms, the release period, contracted rates and stop-sale periods as follows.\n",
    ]
    donem = donem_cumle(obj, "en", tarih_b)
    if donem:
        parcalar.append(f"ARTICLE {n} — CONTRACT PERIOD\n{donem}")
        alt = alt_donem_metin(obj, "en", tarih_b)
        if alt:
            parcalar.append(alt)
        n += 1
    satirlar = []
    for k in obj["oda_kontenjanlari"]:
        extra = f" ({k['aciklama']})" if k.get("aciklama") else ""
        satirlar.append(
            f"- {fmt_sayi(k['adet'], sayi_b)} {metin_oda_tipi(k['oda_tipi'], 'en', k.get('adet'))} rooms{extra}"
        )
    parcalar.append(
        f"ARTICLE {n} — ALLOTMENT\nThe Hotel allocates the following rooms to the Agent:\n"
        + "\n".join(satirlar)
    )
    n += 1
    parcalar.append(f"ARTICLE {n} — RELEASE PERIOD\n{release_cumle(obj, 'en')}")
    n += 1
    parcalar.append(
        f"ARTICLE {n} — RATES\nRates are {_en_rate_unit(obj)}. All rates are quoted in {para}.\n"
        + fiyat_tablo(obj, "en", sayi_b, yapisik)
    )
    n += 1
    ss = stop_sale_metin(obj, "en", tarih_b)
    if ss:
        parcalar.append(f"ARTICLE {n} — STOP-SALE PERIODS\n{ss}")
    return "\n\n".join(parcalar) + "\n"


def sablon_en_confirmation(obj: dict, dil: str, tarih_b: str, sayi_b: str, yapisik: bool) -> str:
    """Commercial allotment and rate confirmation (operator circular)."""
    otel, acente = otel_acente(obj, "en")
    para = (obj.get("meta") or {}).get("para_birimi") or "EUR"
    donem = donem_cumle(obj, "en", tarih_b)
    parcalar = [
        "CONFIDENTIAL — ALLOTMENT AND RATE CONFIRMATION\n",
        f"Hotel: {otel}\nTour operator: {acente}\nCurrency: {para}\n",
    ]
    if donem:
        parcalar.append(f"Season / validity: {donem}")
        alt = alt_donem_metin(obj, "en", tarih_b)
        if alt:
            parcalar.append(alt)
    parcalar.append(
        "1. Inventory commitment\nThe Hotel allocates the inventory below. These rooms are held for the Agent until the release period lapses.\n"
        + kontenjan_tablo(obj, "en", sayi_b)
    )
    parcalar.append(
        f"2. Selling rates\nRates are {_en_rate_unit(obj)} unless a line states otherwise.\n"
        + fiyat_paragraf(obj, "en", sayi_b, yapisik)
    )
    parcalar.append(f"3. Release period\n{release_cumle(obj, 'en')}")
    ss = stop_sale_metin(obj, "en", tarih_b)
    if ss:
        parcalar.append("4. Stop-sale periods\n" + ss)
    parcalar.append("Please sign and return one copy of this confirmation.")
    return "\n\n".join(parcalar) + "\n"


def sablon_en_schedules(obj: dict, dil: str, tarih_b: str, sayi_b: str, yapisik: bool) -> str:
    """Contracting schedules: room categories, rates, release and stop-sale periods."""
    otel, acente = otel_acente(obj, "en")
    donem = donem_cumle(obj, "en", tarih_b) or "as stated in the main agreement"
    gun = obj["release"]["gun"]
    parcalar = [
        f"CONTRACTING SCHEDULES — {otel} / {acente}\nValidity: {donem}\n",
        "SCHEDULE A — ROOM ALLOTMENT\nRoom category | Units committed\n----------------|------------------",
    ]
    for k in obj["oda_kontenjanlari"]:
        extra = f" ({k['aciklama']})" if k.get("aciklama") else ""
        parcalar.append(
            f"{metin_oda_tipi(k['oda_tipi'], 'en', k.get('adet'))}{extra} | {fmt_sayi(k['adet'], sayi_b)}"
        )
    parcalar.append(
        "\nSCHEDULE B — RATES\nRates are per room per night unless a row is marked per person per night.\n"
        + fiyat_tablo(obj, "en", sayi_b, yapisik)
    )
    parcalar.append(
        f"\nSCHEDULE C — RELEASE PERIOD AND STOP-SALE PERIODS\n"
        f"Release period: {gun} days before check-in. {release_cumle(obj, 'en')}"
    )
    ss = stop_sale_metin(obj, "en", tarih_b)
    if ss:
        parcalar.append(ss)
    alt = alt_donem_metin(obj, "en", tarih_b)
    if alt:
        parcalar.append("\nSeasonal rate periods (for Schedule B):\n" + alt)
    return "\n".join(parcalar) + "\n"


SABLONLAR = {
    1: sablon_maddeli,
    2: sablon_tablo,
    3: sablon_paragraf,
    4: sablon_karisik,
    5: sablon_ekler,
    6: sablon_mektup,
}

SABLONLAR_EN = {
    1: sablon_en_articles,
    2: sablon_en_confirmation,
    3: sablon_en_schedules,
}


def bir_ornek(rng: random.Random, seed: int, indeks: int) -> dict:
    dil = "tr" if rng.random() < 0.5 else "en"
    fake = Faker("tr_TR" if dil == "tr" else "en_GB")
    fake.seed_instance(seed * 10007 + indeks * 17)
    obj = asama_a(rng, fake, dil)
    gurultu: list[str] = []

    if rng.random() < 0.15:
        gurultu.append(gurultu_eksik(obj, rng))
    if rng.random() < 0.10:
        etiket = gurultu_tarih(obj, rng)
        if etiket:
            gurultu.append(etiket)
    if rng.random() < 0.10:
        gurultu_oda_uyumsuz(obj, rng)
        gurultu.append("fiyat_oda_uyusmazligi")

    tarih_b = rng.choice(["nokta", "iso", "uzun"])
    sayi_b = rng.choice(["duz", "tr", "en"])
    yapisik = rng.random() < 0.10
    if yapisik:
        gurultu.append("bicim_yapisik")
    if dil == "tr":
        sablon_no = rng.randint(1, 6)
        metin = SABLONLAR[sablon_no](obj, dil, tarih_b, sayi_b, yapisik)
    else:
        sablon_no = rng.randint(1, 3)
        metin = SABLONLAR_EN[sablon_no](obj, dil, tarih_b, sayi_b, yapisik)

    if rng.random() < 0.20:
        metin = metin.rstrip() + "\n" + fazladan_maddeler(obj, dil, sayi_b)
        gurultu.append("fazladan_madde")

    return {
        "metin": metin.strip() + "\n",
        "cikti": obj,
        "meta": {
            "sablon_no": sablon_no,
            "dil": dil,
            "eklenen_gurultu": gurultu,
        },
    }


def jsonl_yaz(yol: Path, ornekler: list[dict]) -> None:
    yol.parent.mkdir(parents=True, exist_ok=True)
    with yol.open("w", encoding="utf-8") as f:
        for o in ornekler:
            f.write(json.dumps(o, ensure_ascii=False) + "\n")


def main() -> None:
    p = argparse.ArgumentParser(description="Sentetik kontenjan sözleşmesi üretir.")
    p.add_argument("--seed", type=int, default=42)
    p.add_argument("--train", type=int, default=320)
    p.add_argument("--val", type=int, default=80)
    p.add_argument("--out-dir", type=Path, default=KOK / "data")
    args = p.parse_args()

    rng = random.Random(args.seed)
    Faker.seed(args.seed)
    toplam = args.train + args.val
    ornekler = [bir_ornek(rng, args.seed, i) for i in range(toplam)]
    jsonl_yaz(args.out_dir / "train.jsonl", ornekler[: args.train])
    jsonl_yaz(args.out_dir / "val.jsonl", ornekler[args.train :])
    print(f"yazıldı {args.out_dir / 'train.jsonl'} ({args.train})")
    print(f"yazıldı {args.out_dir / 'val.jsonl'} ({args.val})")


if __name__ == "__main__":
    main()
