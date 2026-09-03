"use client";

import { useState } from "react";
import type { ExtractionMeta } from "@/lib/graphql-types";
import { confidenceLabel, missingField } from "@/lib/format";

const LOW_CONFIDENCE = 0.75;

export function lookupMeta(
  metas: readonly ExtractionMeta[] | null | undefined,
  path: string,
): ExtractionMeta | undefined {
  return metas?.find((m) => m.alanYolu === path);
}

export function ExtractedField({
  label,
  path,
  value,
  metas,
}: {
  label: string;
  path: string;
  value: string | null | undefined;
  metas?: readonly ExtractionMeta[] | null;
}) {
  const meta = lookupMeta(metas, path);
  const score = meta?.guven ?? null;
  const low = typeof score === "number" && score < LOW_CONFIDENCE;
  const empty = !value;
  const display = empty ? missingField() : value;
  const [draft, setDraft] = useState(display);
  const source = confidenceLabel(meta?.kaynakSayfa, score);

  return (
    <div
      className={`border-b-[0.5px] border-[var(--line)] px-3 py-2 last:border-0 ${
        low ? "bg-[var(--yellow-bg)]" : ""
      }`}
    >
      <div className="flex items-baseline justify-between gap-3">
        <span className="text-[13px] text-[var(--muted)]">{label}</span>
        {source ? (
          <span className="text-[11px] text-[var(--muted)]">{source}</span>
        ) : null}
      </div>
      {low && !empty ? (
        <div className="mt-1">
          <label htmlFor={`alan-${path}`} className="sr-only">
            {label}
          </label>
          <input
            id={`alan-${path}`}
            value={draft}
            onChange={(e) => setDraft(e.target.value)}
          />
          <p className="mt-1 text-[12px] text-[var(--yellow-ink)]">
            Düşük güven, kontrol edin
          </p>
        </div>
      ) : (
        <p className="mt-0.5 text-[14px]">{display}</p>
      )}
    </div>
  );
}
