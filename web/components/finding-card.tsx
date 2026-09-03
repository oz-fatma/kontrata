export function FindingCard({
  tone,
  title,
  body,
}: {
  tone: "red" | "yellow";
  title: string;
  body: string;
}) {
  const bar = tone === "red" ? "var(--red)" : "var(--yellow)";
  const bg = tone === "red" ? "var(--red-bg)" : "var(--yellow-bg)";
  return (
    <div
      className="mb-2 px-3 py-2 text-[13px]"
      style={{
        background: bg,
        borderLeft: `3px solid ${bar}`,
        borderRadius: 0,
      }}
    >
      <p className="font-medium">{title}</p>
      <p className="mt-0.5 text-[12px] text-[var(--muted)]">{body}</p>
    </div>
  );
}
