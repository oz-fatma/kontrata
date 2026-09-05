# -*- coding: utf-8 -*-
"""Dorduncu test sozlesmesi.

Argos'tan daha karmasik (gercek 2 alt donem var), TUI'den daha basit
(tek sayfa, tek fiyat pansiyonu, az stop-sale). Chunking'in "orta"
zorlukta ne yaptigini olcmek icin.

Beklenen cikti:
  donem: 2026-05-01 - 2026-09-30, 2 alt donem (Haziran-Temmuz yuksek,
         digerleri normal)
  oda_kontenjanlari: standart 80, aile 25, suit 10  (3 kalem)
  fiyatlar: 3 oda tipi x 2 donem = 6 satir, hepsi BB, EUR
  release: 7 gun, isim_listesi
  stop_sale: 1 aralik (15-20 Temmuz, tum tipler)
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
    P("KONTENJAN SÖZLEŞMESİ", "h1"),
    P("Anex Tour Turizm A.Ş. (Operatör) ile Belek Garden Resort (Tesis) "
      "arasında aşağıdaki şartlarla akdedilmiştir.", "p"),
    Spacer(1, 6),

    P("MADDE 1 — DÖNEM", "h2"),
    P("Sözleşme 01.05.2026 tarihinde başlar, 30.09.2026 tarihinde sona "
      "erer. 01.06.2026 – 15.07.2026 arası \u201cYüksek Sezon\u201d, kalan "
      "süre \u201cNormal Sezon\u201d olarak uygulanır.", "p"),

    P("MADDE 2 — ODA KONTENJANI", "h2"),
    tbl([
        ["Oda tipi", "Adet"],
        ["Standart", "80"],
        ["Aile", "25"],
        ["Suit", "10"],
    ], [70 * mm, 40 * mm]),

    P("MADDE 3 — FİYATLAR", "h2"),
    P("Kişi başı gecelik, yarım pansiyon (BB), EUR cinsinden.", "p"),
    tbl([
        ["Oda tipi", "Normal Sezon", "Yüksek Sezon"],
        ["Standart", "38,00", "55,00"],
        ["Aile", "52,00", "74,00"],
        ["Suit", "89,00", "126,00"],
    ], [55 * mm, 45 * mm, 45 * mm]),

    P("MADDE 4 — RELEASE", "h2"),
    P("Operatör, satılmayan kontenjanı misafir girişinden 7 gün önce "
      "isim listesiyle birlikte tesise bildirir.", "p"),

    P("MADDE 5 — STOP-SALE", "h2"),
    P("Tesis, 15.07.2026 – 20.07.2026 tarihleri arasında tüm oda "
      "tiplerinde satışı durdurma hakkını saklı tutar.", "p"),

    P("MADDE 6 — YETKİLİ MAHKEME", "h2"),
    P("Uyuşmazlıklarda Antalya mahkemeleri yetkilidir.", "p"),

    Spacer(1, 14),
    P("Anex Tour Turizm A.Ş.&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;"
      "&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;Belek Garden Resort",
      "small"),
]

doc = SimpleDocTemplate(
    "/home/claude/pdfs2/anex-belek.pdf", pagesize=A4,
    leftMargin=22 * mm, rightMargin=22 * mm,
    topMargin=20 * mm, bottomMargin=20 * mm,
    title="Anex Tour - Belek Garden Resort kontenjan sözleşmesi")

import os
os.makedirs("/home/claude/pdfs2", exist_ok=True)
doc.build(story)
print("yazildi: anex-belek.pdf")
