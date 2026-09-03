"use client";

import { useEffect, useState } from "react";
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
  lines,
  metas,
  readOnly,
  onSave,
}: {
  label: string;
  path: string;
  lines: string[];
  metas?: readonly ExtractionMeta[] | null;
  readOnly?: boolean;
  onSave?: (path: string, value: string) => Promise<void>;
}) {
  const meta = lookupMeta(metas, path);
  const score = meta?.guven ?? null;
  const low = typeof score === "number" && score < LOW_CONFIDENCE;
  const displayLines = lines.length > 0 ? lines : [missingField()];
  const empty = lines.length === 0;
  const joined = displayLines.join("\n");
  const [draft, setDraft] = useState(joined);
  const [saving, setSaving] = useState(false);
  const source = confidenceLabel(meta?.kaynakSayfa, score);
  const editable = Boolean(low && !empty && !readOnly && onSave);

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
      className={`border-b-[0.5px] border-[var(--line)] px-3 py-2 last:border-0 ${
        low && !readOnly ? "bg-[var(--yellow-bg)]" : ""
      }`}
    >
      <div className="flex items-baseline justify-between gap-3">
        <span className="text-[13px] text-[var(--muted)]">{label}</span>
        {source ? (
          <span className="text-[11px] text-[var(--muted)]">{source}</span>
        ) : null}
      </div>
      {editable ? (
        <div className="mt-1">
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
        <div className="mt-0.5 text-[14px]">
          {displayLines.map((line, i) => (
            <div key={`${path}-${i}`} className="block">
              {line}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
