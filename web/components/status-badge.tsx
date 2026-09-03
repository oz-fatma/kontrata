import type { StatusTone } from "@/lib/format";

const toneClass: Record<StatusTone, string> = {
  muted: "text-[var(--muted)] border-[var(--line)] bg-white",
  blue: "text-[var(--blue)] border-[var(--blue)]/30 bg-[var(--blue-bg)]",
  yellow: "text-[var(--yellow-ink)] border-[var(--yellow)]/40 bg-[var(--yellow-bg)]",
  green: "text-[var(--green)] border-[var(--green)]/30 bg-[var(--green-bg)]",
  red: "text-[var(--red)] border-[var(--red)]/30 bg-[var(--red-bg)]",
};

export function StatusBadge({
  label,
  tone,
}: {
  label: string;
  tone: StatusTone;
}) {
  return (
    <span
      className={`inline-flex items-center rounded-control border px-2 py-0.5 text-[12px] ${toneClass[tone]}`}
    >
      {label}
    </span>
  );
}
