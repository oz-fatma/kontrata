# Mimari

Kontrata üç süreçten oluşur: Electron kabuğu, gömülü Go API, tesisteki MongoDB.
Çıkarım HuggingFace Inference Endpoint'e gider; yolda maskeleme zorunludur.

## Bileşenler

```mermaid
flowchart LR
  subgraph tesis [Tesis / kullanıcı makinesi]
    UI[Next.js static export]
    EL[Electron]
    API[Go API / GraphQL]
    FS[(UPLOAD_DIR PDF)]
    DB[(MongoDB replica set)]
    UI --> EL
    EL --> API
    API --> FS
    API --> DB
  end
  subgraph dis [Tesis dışı]
    LLM[HuggingFace Inference Endpoint]
  end
  API -->|"yalnızca maskelenmiş metin"| LLM
  LLM -->|JSON adayı| API
```

| Bileşen | Görev |
| --- | --- |
| `web/` | Liste, detay, ayarlar, yönetici paneli. `output: 'export'` |
| `desktop/` | Pencere, `kontrata://` şema, API alt süreç, `safeStorage` ayarları |
| `backend/` | Chi + gqlgen. Kimlik, sözleşme, çıkarım kuyruğu, yönlendirici |
| MongoDB | Hesap, sözleşme, denetim, prompt, `llm_cagrilari` |
| HF uç 1 / uç 2 | Fine-tune Qwen2.5-1.5B. Yönlendirici `aktifIstek` ile seçer |

## Veri akışı

```mermaid
sequenceDiagram
  participant K as Kullanıcı
  participant UI as Arayüz
  participant API as Go API
  participant Disk as UPLOAD_DIR
  participant DB as MongoDB
  participant M as mask.Apply
  participant LLM as HF Endpoint

  K->>UI: PDF yükle
  UI->>API: sozlesmeYukle
  API->>Disk: UUID adıyla yaz
  API->>DB: sozlesmeler YUKLENDI
  API-->>UI: id (Sırada)
  Note over API: İşçi ISLENIYOR
  API->>Disk: PDF oku, metin çıkar
  API->>M: sözleşme metni
  M-->>API: [EPOSTA] / [TELEFON] / [TCKN]
  API->>LLM: Okuyucu (maskeli)
  LLM-->>API: JSON adayı
  API->>API: RepairJSON + Normalize + şema
  API->>LLM: Denetçi (maskeli, isteğe bağlı)
  API->>DB: alanlar, kural bulguları, llm_cagrilari
  API-->>UI: INCELENMEYI_BEKLIYOR
```

Maskeleme, yönetici prompt'undan ve `maxToken` ayarından bağımsızdır.
`llm_cagrilari` süre, uç, agent, hata tipi tutar; gövde tutmaz.

Hesap silinince Mongo kayıtları transaction içinde kalkar; PDF'ler commit
sonrası diskten silinir. Denetim satırları `silinmis` olarak kalır.
Ayrıntı: [`kvkk.md`](kvkk.md).
