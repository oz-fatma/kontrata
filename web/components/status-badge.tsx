import {
  AlertCircle,
  CheckCircle2,
  Clock,
  Loader2,
  type LucideIcon,
} from "lucide-react";
import type { SozlesmeDurumu as ContractStatus } from "@/generated/graphql";
import { SozlesmeDurumu } from "@/lib/enums";
import type { StatusTone } from "@/lib/format";

const toneClass: Record<StatusTone, string> = {
  muted: "text-[var(--ink-muted)] border-[var(--border)] bg-[var(--surface)]",
  blue: "text-[var(--blue)] border-[var(--blue)]/30 bg-[var(--blue-bg)]",
  yellow: "text-[var(--yellow-ink)] border-[var(--warning)]/40 bg-[var(--yellow-bg)]",
  green: "text-[var(--green)] border-[var(--green)]/30 bg-[var(--green-bg)]",
  red: "text-[var(--red)] border-[var(--red)]/30 bg-[var(--red-bg)]",
};

const statusIcon: Partial<Record<ContractStatus, LucideIcon>> = {
  [SozlesmeDurumu.Yuklendi]: Clock,
  [SozlesmeDurumu.Isleniyor]: Loader2,
  [SozlesmeDurumu.IncelenmeyiBekliyor]: Clock,
  [SozlesmeDurumu.Onaylandi]: CheckCircle2,
  [SozlesmeDurumu.Hata]: AlertCircle,
};

export function StatusBadge({
  label,
  tone,
  durum,
}: {
  label: string;
  tone: StatusTone;
  durum?: ContractStatus | string;
}) {
  const Icon = durum ? statusIcon[durum as ContractStatus] : undefined;
  const spinning = durum === SozlesmeDurumu.Isleniyor;

  return (
    <span
      className={`inline-flex items-center gap-1 rounded-control border px-2 py-0.5 text-[12px] font-medium ${toneClass[tone]}`}
    >
      {Icon ? (
        <Icon
          className={`h-3.5 w-3.5 shrink-0 ${spinning ? "animate-spin" : ""}`}
          strokeWidth={2}
          aria-hidden
        />
      ) : null}
      {label}
    </span>
  );
}
