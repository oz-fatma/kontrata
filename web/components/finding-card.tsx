export function FindingCard({
  tone,
  title,
  body,
  source,
}: {
  tone: "red" | "yellow" | "muted";
  title: string;
  body: string;
  source?: "kural" | "model";
}) {
  const bar =
    tone === "red" ? "var(--red)" : tone === "yellow" ? "var(--warning)" : "var(--ink-muted)";
  const bg =
    tone === "red" ? "var(--red-bg)" : tone === "yellow" ? "var(--yellow-bg)" : "var(--surface-subtle)";
  return (
    <div
      className="mb-2 rounded-control px-3 py-2 text-[13px]"
      style={{
        background: bg,
        borderLeft: `3px solid ${bar}`,
      }}
    >
      <div className="flex items-start justify-between gap-2">
        <p className="font-medium text-[var(--ink)]">{title}</p>
        {source ? (
          <span className="meta-text inline-flex shrink-0 items-center rounded-control border-[0.5px] border-[var(--border)] bg-[var(--surface)] px-1.5 py-0.5 text-[11px]">
            {source}
          </span>
        ) : null}
      </div>
      <p className="meta-text mt-0.5">{body}</p>
    </div>
  );
}
