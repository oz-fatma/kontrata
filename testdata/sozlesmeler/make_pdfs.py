# -*- coding: utf-8 -*-
"""Kontrata test PDF'leri.

Üç senaryo:
  1. argos-megep.pdf      — MEGEP örnek sözleşmesi (bilinen doğru çıktı,
                             ml/schema/ornek-argos.json ile karşılaştırılabilir)
  2. tui-2026-yaz.pdf     — modern operatör kontratı, release + stop-sale var,
                             alt dönemler ve tablo biçiminde fiyatlar
  3. coral-bozuk.pdf      — kasıtlı tutarsızlıklar: fiyat tablosunda kontenjanda
                             olmayan oda tipi, çelişkili tarih, eksik stop-sale
"""
from reportlab.lib.pagesizes import A4
from reportlab.lib.units import mm
from reportlab.lib import colors
from reportlab.lib.styles import ParagraphStyle
from reportlab.pdfbase import pdfmetrics
from reportlab.pdfbase.ttfonts import TTFont
from reportlab.platypus import (SimpleDocTemplate, Paragraph, Spacer, Table,
                                TableStyle, PageBreak)

D = "/usr/share/fonts/truetype/dejavu/"
pdfmetrics.registerFont(TTFont("DJ", D + "DejaVuSans.ttf"))
pdfmetrics.registerFont(TTFont("DJ-B", D + "DejaVuSans-Bold.ttf"))

st = {
    "h1": ParagraphStyle("h1", fontName="DJ-B", fontSize=13, leading=17,
                         spaceAfter=10, alignment=1),
    "h2": ParagraphStyle("h2", fontName="DJ-B", fontSize=10, leading=14,
                         spaceBefore=10, spaceAfter=5),
    "p": ParagraphStyle("p", fontName="DJ", fontSize=9.5, leading=14,
                        spaceAfter=5),
    "small": ParagraphStyle("small", fontName="DJ", fontSize=8.5, leading=12,
                            textColor=colors.HexColor("#555555")),
}


def P(t, s="p"):
    return Paragraph(t, st[s])


def tbl(rows, widths):
    t = Table(rows, colWidths=widths, repeatRows=1)
    t.setStyle(TableStyle([
        ("FONTNAME", (0, 0), (-1, 0), "DJ-B"),
        ("FONTNAME", (0, 1), (-1, -1), "DJ"),
        ("FONTSIZE", (0, 0), (-1, -1), 9),
        ("TOPPADDING", (0, 0), (-1, -1), 4),
        ("BOTTOMPADDING", (0, 0), (-1, -1), 4),
        ("GRID", (0, 0), (-1, -1), 0.4, colors.HexColor("#999999")),
        ("BACKGROUND", (0, 0), (-1, 0), colors.HexColor("#eeeeee")),
        ("VALIGN", (0, 0), (-1, -1), "MIDDLE"),
    ]))
    return t


def build(path, story, title):
    doc = SimpleDocTemplate(path, pagesize=A4,
                            leftMargin=22 * mm, rightMargin=22 * mm,
                            topMargin=20 * mm, bottomMargin=20 * mm,
                            title=title)
    doc.build(story)
    print("yazildi:", path)


# ------------------------------------------------------------------ 1
# MEGEP Argos örneği — ml/schema/ornek-argos.json ile birebir eşleşmeli
argos = [
    P("HİZMET SÖZLEŞMESİ", "h1"),
    P("Side Turizm Seyahat Acentesi ile Argos Otel arasında aşağıdaki "
      "koşullarda kontenjan sözleşmesi akdedilmiştir.", "p"),
    Spacer(1, 6),

    P("MADDE 1 — SÖZLEŞME SÜRESİ", "h2"),
    P("İşbu sözleşme 01.04.2026 tarihinde yürürlüğe girer ve 31.10.2026 "
      "tarihinde sona erer.", "p"),

    P("MADDE 2 — ODA KONTENJANI", "h2"),
    P("Otel, acenteye 170 normal oda, 20 suit oda, 5 balayı odası ve "
      "1 özürlü odası tahsis etmeyi kabul eder.", "p"),

    P("MADDE 3 — İSİM LİSTESİ", "h2"),
    P("Acente, oda kontenjanı ile ilgili son durumu müşterilerin otele "
      "girişini 10 gün önce belirler ve isim listesi ile birlikte otele "
      "bildirir.", "p"),

    P("MADDE 4 — NO SHOW", "h2"),
    P("Acente, sözleşme süresi içerisinde doğabilecek iptallerden dolayı "
      "otelin no show talebini karşılar.", "p"),

    P("MADDE 5 — SHORT DURUMU", "h2"),
    P("Otelin short'a düşmesi ve bu durumda acente müşterisinin zarar "
      "görmesi halinde doğabilecek zararlardan otel sorumludur.", "p"),

    P("MADDE 6 — FATURALAMA", "h2"),
    P("Faturalar, müşterinin otele giriş günündeki Türkiye Cumhuriyet "
      "Merkez Bankası kuru esas alınarak İngiliz Sterlini üzerinden "
      "düzenlenir.", "p"),

    P("MADDE 7 — YETKİLİ MAHKEME", "h2"),
    P("İşbu sözleşmeden doğacak anlaşmazlıklarda Antalya mahkemeleri ve "
      "icra daireleri yetkilidir.", "p"),

    P("MADDE 8 — ODA FİYATLARI", "h2"),
    tbl([
        ["Oda tipi", "Gecelik ücret (GBP)"],
        ["Tek kişilik", "50"],
        ["İki kişilik", "80"],
        ["Üç kişilik", "95"],
        ["Suit", "110"],
        ["Balayı", "130"],
        ["Özürlü", "80"],
    ], [70 * mm, 55 * mm]),

    Spacer(1, 14),
    P("Side Turizm Seyahat Acentesi&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;"
      "&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;Argos Otel", "small"),
]

# ------------------------------------------------------------------ 2
# Modern operatör kontratı — release, stop-sale, alt dönemler
tui = [
    P("KONTENJAN TAHSİS SÖZLEŞMESİ", "h1"),
    P("Taraflar: TUI Turizm A.Ş. (Operatör) — Kemer Resort Hotel (Tesis)", "p"),
    P("Sözleşme no: TUI-KMR-2026-S01 &nbsp;·&nbsp; Düzenleme tarihi: 12.01.2026",
      "small"),
    Spacer(1, 8),

    P("1. SÖZLEŞME DÖNEMİ", "h2"),
    P("İşbu sözleşme 15 Nisan 2026 tarihinde başlar, 25 Ekim 2026 tarihinde "
      "sona erer. Sezon içi fiyat dönemleri aşağıdaki gibidir:", "p"),
    tbl([
        ["Dönem adı", "Başlangıç", "Bitiş"],
        ["Erken sezon", "15.04.2026", "31.05.2026"],
        ["Yüksek sezon", "01.06.2026", "15.09.2026"],
        ["Geç sezon", "16.09.2026", "25.10.2026"],
    ], [50 * mm, 40 * mm, 40 * mm]),

    P("2. ODA KONTENJANI", "h2"),
    P("Tesis, operatöre aşağıdaki oda kontenjanını tahsis eder:", "p"),
    tbl([
        ["Oda tipi", "Adet"],
        ["Standart", "240"],
        ["Aile odası", "45"],
        ["Deluxe deniz manzaralı", "30"],
        ["Suit", "12"],
    ], [70 * mm, 35 * mm]),

    P("3. RELEASE SÜRESİ", "h2"),
    P("Operatör, satılmayan kontenjanı misafir girişinden en geç 14 gün önce "
      "tesise iade etmekle yükümlüdür. Bu süre içinde iade edilmeyen odalar "
      "operatör hesabına yazılır.", "p"),

    PageBreak(),

    P("4. FİYATLAR", "h2"),
    P("Aşağıdaki fiyatlar kişi başı gecelik, her şey dahil (AI) pansiyon "
      "esasına göre EUR cinsindendir.", "p"),
    tbl([
        ["Oda tipi", "Erken sezon", "Yüksek sezon", "Geç sezon"],
        ["Standart", "48,00", "72,50", "52,00"],
        ["Aile odası", "61,00", "94,00", "66,50"],
        ["Deluxe deniz manzaralı", "75,00", "118,00", "82,00"],
        ["Suit", "112,00", "165,00", "124,00"],
    ], [50 * mm, 32 * mm, 32 * mm, 32 * mm]),

    P("5. SATIŞ DURDURMA (STOP-SALE)", "h2"),
    P("Tesis, aşağıdaki tarihlerde satışı durdurma hakkını saklı tutar. "
      "Bildirim yazılı olarak yapılır.", "p"),
    tbl([
        ["Başlangıç", "Bitiş", "Kapsam"],
        ["10.07.2026", "18.07.2026", "Tüm oda tipleri"],
        ["20.08.2026", "24.08.2026", "Suit"],
    ], [42 * mm, 42 * mm, 50 * mm]),

    P("6. ÇOCUK POLİTİKASI", "h2"),
    P("0-2 yaş ücretsiz. 3-6 yaş %75 indirimli. 7-12 yaş %50 indirimli. "
      "İki yetişkin yanında en fazla iki çocuk kabul edilir.", "p"),

    P("7. ÖDEME KOŞULLARI", "h2"),
    P("Faturalar, düzenleme tarihinden itibaren 30 gün içinde ödenir. "
      "Gecikme halinde aylık %2 gecikme faizi uygulanır.", "p"),

    P("8. İPTAL KOŞULLARI", "h2"),
    P("Grup rezervasyonlarında iptal, girişten 21 gün önce bildirilmelidir. "
      "Bireysel rezervasyonlarda bu süre 7 gündür.", "p"),

    Spacer(1, 12),
    P("TUI Turizm A.Ş.&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;"
      "&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;Kemer Resort Hotel", "small"),
]

# ------------------------------------------------------------------ 3
# Kasıtlı bozuk — Denetçi agent'ın bulması gerekenler
coral = [
    P("ALLOTMENT CONTRACT", "h1"),
    P("Coral Travel — Belek Palace Hotel", "p"),
    Spacer(1, 8),

    P("ARTICLE 1 — CONTRACT PERIOD", "h2"),
    P("This contract is valid from 01.05.2026 until 20.04.2026.", "p"),

    P("ARTICLE 2 — ROOM ALLOTMENT", "h2"),
    P("The Hotel allocates the following rooms to the Agent:", "p"),
    tbl([
        ["Room type", "Quantity"],
        ["Standard", "150"],
        ["Family", "40"],
    ], [70 * mm, 35 * mm]),

    P("ARTICLE 3 — NAME LIST", "h2"),
    P("The Agent shall submit the final name list approximately 10 days "
      "prior to guest arrival.", "p"),

    P("ARTICLE 4 — RATES", "h2"),
    P("Rates are per room per night in EUR, bed and breakfast basis.", "p"),
    tbl([
        ["Room type", "Rate"],
        ["Standard", "65,00"],
        ["Family", "98,00"],
        ["Junior suite", "140,00"],
        ["Penthouse", "260,00"],
    ], [70 * mm, 35 * mm]),

    P("ARTICLE 5 — PAYMENT", "h2"),
    P("Invoices shall be settled within 45 days of issue.", "p"),

    P("ARTICLE 6 — JURISDICTION", "h2"),
    P("Antalya courts shall have jurisdiction over any dispute.", "p"),

    Spacer(1, 12),
]

build("/home/claude/pdfs/argos-megep.pdf", argos, "Argos Otel kontenjan sözleşmesi")
build("/home/claude/pdfs/tui-2026-yaz.pdf", tui, "TUI Kemer Resort 2026 yaz kontratı")
build("/home/claude/pdfs/coral-bozuk.pdf", coral, "Coral Travel Belek Palace allotment")
