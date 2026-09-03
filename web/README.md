# Kontrata arayüz

Next.js 15 (App Router) static export. Electron kabuğu paketlenmiş çıktıyı `web/out` dizininden yükler (`desktop/README.md`).

## Çalıştırma

Tarayıcıda geliştirme için API'nin `http://localhost:8080/graphql` adresinde ayakta olması gerekir.

```bash
cd web
cp .env.example .env.local   # isteğe bağlı; varsayılan localhost
npm install
npm run codegen
npm run dev
```

Electron `npm run dev` ile açıldığında arayüz yine `localhost:3000` yükler; GraphQL istekleri preload üzerinden `http://127.0.0.1:17890` adresine gider.

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

Erişim jetonu bellektedir. Yenileme jetonu tarayıcıda `sessionStorage`, Electron'da `safeStorage` (preload IPC) ile saklanır.
