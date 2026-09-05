"use client";

import { useEffect, useState } from "react";
import type { ExtractionMeta } from "@/lib/graphql-types";
import { confidenceLabel, missingField } from "@/lib/format";

/** Düzenleme ve uyarı için tek eşik */
const REVIEW_CONFIDENCE = 0.85;

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
  const empty = lines.length === 0;
  const needsReview =
    empty || (typeof score === "number" && score < REVIEW_CONFIDENCE);
  const displayLines = empty ? [missingField()] : lines;
  const joined = empty ? "" : lines.join("\n");
  const [draft, setDraft] = useState(joined);
  const [saving, setSaving] = useState(false);
  const source = confidenceLabel(meta?.kaynakSayfa, score);
  const editable = Boolean(needsReview && !readOnly && onSave);
  const multiLine = listStyle || (!empty && lines.length > 1);

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
        needsReview && !readOnly ? "bg-[var(--yellow-bg)]" : ""
      }`}
    >
      <div className="flex items-baseline justify-between gap-3">
        <span className="inline-flex items-center gap-1.5 meta-text">
          {needsReview && !readOnly ? (
            <span
              className="inline-block h-1.5 w-1.5 shrink-0 rounded-full bg-[var(--warning)]"
              title={empty ? "Eksik alan" : "Düşük güven skoru"}
              aria-hidden
            />
          ) : null}
          {label}
        </span>
        {source ? (
          <span
            className={`tabular-nums text-[11px] font-medium ${
              needsReview ? "text-[var(--yellow-ink)]" : "text-[var(--ink-muted)]"
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
          {multiLine ? (
            <textarea
              id={`alan-${path}`}
              value={draft}
              rows={Math.min(8, Math.max(3, empty ? 3 : lines.length))}
              disabled={saving}
              placeholder={missingField()}
              onChange={(e) => setDraft(e.target.value)}
              onBlur={() => void commit()}
            />
          ) : (
            <input
              id={`alan-${path}`}
              value={draft}
              disabled={saving}
              placeholder={missingField()}
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
            {empty ? "Eksik alan, doldurun" : "Düşük güven, kontrol edin"}
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
