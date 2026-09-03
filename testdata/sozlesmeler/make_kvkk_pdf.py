# -*- coding: utf-8 -*-
"""KVKK maskeleme testi icin sozlesme.

Argos sozlesmesinin kisisel veri iceren hali. Metinde bilerek birakilan
veriler:
  - 2 e-posta adresi
  - 2 telefon numarasi (05xx ve +90 bicimi)
  - 1 TCKN (11 haneli)

Beklenen: bu PDF yuklendiginde backend logunda "maskelendi=5" gorunmeli
ve LLM'e giden metinde bu degerler bulunmamali.
"""
from reportlab.lib.pagesizes import A4
from reportlab.lib.units import mm
from reportlab.lib import colors
from reportlab.lib.styles import ParagraphStyle
from reportlab.pdfbase import pdfmetrics
from reportlab.pdfbase.ttfonts import TTFont
from reportlab.platypus import SimpleDocTemplate, Paragraph, Spacer, Table, TableStyle

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
    ]))
    return t


story = [
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

    P("MADDE 4 — YETKİLİ KİŞİLER VE İLETİŞİM", "h2"),
    P("Otel adına yetkili kişi: Ayşe Demir, T.C. kimlik no 12345678901. "
      "Rezervasyon bildirimleri ayse.demir@argosotel.com adresine ve "
      "0532 445 67 89 numaralı telefona yapılır.", "p"),
    P("Acente adına yetkili kişi: Mehmet Yıldız. İletişim: "
      "m.yildiz@sideturizm.com.tr, +90 242 813 45 67.", "p"),

    P("MADDE 5 — FATURALAMA", "h2"),
    P("Faturalar, müşterinin otele giriş günündeki Türkiye Cumhuriyet "
      "Merkez Bankası kuru esas alınarak İngiliz Sterlini üzerinden "
      "düzenlenir.", "p"),

    P("MADDE 6 — YETKİLİ MAHKEME", "h2"),
    P("İşbu sözleşmeden doğacak anlaşmazlıklarda Antalya mahkemeleri "
      "yetkilidir.", "p"),

    P("MADDE 7 — ODA FİYATLARI", "h2"),
    tbl([
        ["Oda tipi", "Gecelik ücret (GBP)"],
        ["Tek kişilik", "50"],
        ["İki kişilik", "80"],
        ["Suit", "110"],
        ["Balayı", "130"],
    ], [70 * mm, 55 * mm]),

    Spacer(1, 14),
    P("Side Turizm Seyahat Acentesi&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;"
      "&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;Argos Otel", "small"),
]

doc = SimpleDocTemplate(
    "/home/claude/pdfs/argos-kisisel-veri.pdf", pagesize=A4,
    leftMargin=22 * mm, rightMargin=22 * mm,
    topMargin=20 * mm, bottomMargin=20 * mm,
    title="Argos Otel kontenjan sözleşmesi (kişisel veri içerir)")
doc.build(story)
print("yazildi: argos-kisisel-veri.pdf")
