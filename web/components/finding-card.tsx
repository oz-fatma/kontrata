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
    tone === "red" ? "var(--red)" : tone === "yellow" ? "var(--yellow)" : "var(--muted)";
  const bg =
    tone === "red" ? "var(--red-bg)" : tone === "yellow" ? "var(--yellow-bg)" : "var(--subtle)";
  return (
    <div
      className="mb-2 px-3 py-2 text-[13px]"
      style={{
        background: bg,
        borderLeft: `3px solid ${bar}`,
        borderRadius: 0,
      }}
    >
      <div className="flex items-start justify-between gap-2">
        <p className="font-medium">{title}</p>
        {source ? (
          <span className="inline-flex shrink-0 items-center rounded-control border-[0.5px] border-[var(--line)] bg-white px-1.5 py-0.5 text-[11px] text-[var(--muted)]">
            {source}
          </span>
        ) : null}
      </div>
      <p className="mt-0.5 text-[12px] text-[var(--muted)]">{body}</p>
    </div>
  );
}
