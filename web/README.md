# Kontrata arayüz

Next.js 15 (App Router) static export. Electron kabuğuna Aşama 10'da gömülecek.

## Çalıştırma

API'nin `http://localhost:8080/graphql` adresinde ayakta olması gerekir.

```bash
cd web
cp .env.example .env.local   # isteğe bağlı; varsayılan localhost
npm install
npm run codegen
npm run dev
```

Üretim derlemesi `out/` dizinine statik dosya üretir:

```bash
npm run build
```

## Betikler

- `npm run codegen` — `backend/graph/schema.graphqls` ve `graphql/*.graphql` dosyalarından TypeScript üretir
- `npm run dev` — geliştirme sunucusu
- `npm run build` — codegen + static export
- `npm run lint` — ESLint

## Oturum

Erişim jetonu bellekte, yenileme jetonu `sessionStorage` içindedir (geçici; Aşama 10'da Electron güvenli deposuna taşınır).
