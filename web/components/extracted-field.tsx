"use client";

import { useEffect, useState } from "react";
import type { ExtractionMeta } from "@/lib/graphql-types";
import { confidenceLabel, missingField } from "@/lib/format";

const LOW_CONFIDENCE = 0.75;
const WARN_CONFIDENCE = 0.85;

export function lookupMeta(
  metas: readonly ExtractionMeta[] | null | undefined,
  path: string,
): ExtractionMeta | undefined {
  return metas?.find((m) => m.alanYolu === path);
}

export function ExtractedField({
  label,
  path,
  lines,
  metas,
  readOnly,
  onSave,
  listStyle,
}: {
  label: string;
  path: string;
  lines: string[];
  metas?: readonly ExtractionMeta[] | null;
  readOnly?: boolean;
  onSave?: (path: string, value: string) => Promise<void>;
  /** Fiyat, kontenjan ve stop-sale gibi çok satırlı listeler */
  listStyle?: boolean;
}) {
  const meta = lookupMeta(metas, path);
  const score = meta?.guven ?? null;
  const low = typeof score === "number" && score < LOW_CONFIDENCE;
  const warn = typeof score === "number" && score < WARN_CONFIDENCE;
  const displayLines = lines.length > 0 ? lines : [missingField()];
  const empty = lines.length === 0;
  const joined = displayLines.join("\n");
  const [draft, setDraft] = useState(joined);
  const [saving, setSaving] = useState(false);
  const source = confidenceLabel(meta?.kaynakSayfa, score);
  const editable = Boolean(low && !empty && !readOnly && onSave);
  const multiLine = listStyle || displayLines.length > 1;

  useEffect(() => {
    setDraft(joined);
  }, [joined]);

  async function commit() {
    if (!onSave || draft === joined) {
      return;
    }
    setSaving(true);
    try {
      await onSave(path, draft);
    } finally {
      setSaving(false);
    }
  }

  return (
    <div
      className={`border-b-[0.5px] border-[var(--border)] px-[var(--space-card)] py-3 last:border-0 ${
        low && !readOnly ? "bg-[var(--yellow-bg)]" : ""
      }`}
    >
      <div className="flex items-baseline justify-between gap-3">
        <span className="inline-flex items-center gap-1.5 meta-text">
          {warn ? (
            <span
              className="inline-block h-1.5 w-1.5 shrink-0 rounded-full bg-[var(--warning)]"
              title="Düşük güven skoru"
              aria-hidden
            />
          ) : null}
          {label}
        </span>
        {source ? (
          <span
            className={`tabular-nums text-[11px] font-medium ${
              warn ? "text-[var(--yellow-ink)]" : "text-[var(--ink-muted)]"
            }`}
          >
            {source}
          </span>
        ) : null}
      </div>
      {editable ? (
        <div className="mt-2">
          <label htmlFor={`alan-${path}`} className="sr-only">
            {label}
          </label>
          {displayLines.length > 1 ? (
            <textarea
              id={`alan-${path}`}
              value={draft}
              rows={Math.min(8, Math.max(3, displayLines.length))}
              disabled={saving}
              onChange={(e) => setDraft(e.target.value)}
              onBlur={() => void commit()}
            />
          ) : (
            <input
              id={`alan-${path}`}
              value={draft}
              disabled={saving}
              onChange={(e) => setDraft(e.target.value)}
              onBlur={() => void commit()}
              onKeyDown={(e) => {
                if (e.key === "Enter") {
                  e.currentTarget.blur();
                }
              }}
            />
          )}
          <p className="mt-1 text-[12px] text-[var(--yellow-ink)]">
            Düşük güven, kontrol edin
          </p>
        </div>
      ) : (
        <div className={`mt-1 text-[14px] leading-normal ${multiLine ? "field-lines" : ""}`}>
          {displayLines.map((line, i) => (
            <div key={`${path}-${i}`} className={multiLine ? "field-line tabular-nums" : "tabular-nums"}>
              {line}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
