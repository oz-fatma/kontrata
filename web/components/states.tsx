import { Loader2 } from "lucide-react";
import type { LucideIcon } from "lucide-react";
import type { ReactNode } from "react";

export function LoadingState({ label = "Yükleniyor" }: { label?: string }) {
  return (
    <div
      className="flex items-center justify-center gap-2 py-8 text-[14px] font-medium text-[var(--ink-muted)]"
      role="status"
    >
      <Loader2
        className="h-3.5 w-3.5 shrink-0 animate-spin text-[var(--blue)]"
        strokeWidth={2}
        aria-hidden
      />
      <span>{label}…</span>
    </div>
  );
}

export function EmptyState({
  title,
  detail,
  icon: Icon,
  compact,
}: {
  title: string;
  detail?: string;
  icon?: LucideIcon;
  compact?: boolean;
}) {
  const inner = (
    <>
      {Icon ? (
        <Icon
          className="mx-auto mb-3 h-7 w-7 text-[var(--ink-muted)]"
          strokeWidth={1.5}
          aria-hidden
        />
      ) : null}
      <p className={`${compact ? "text-[14px]" : "text-[15px]"} font-semibold text-[var(--ink)]`}>
        {title}
      </p>
      {detail ? <p className="meta-text mt-1">{detail}</p> : null}
    </>
  );

  if (compact) {
    return <div className="py-4 text-center">{inner}</div>;
  }

  return (
    <div className="card px-[var(--space-card)] py-10 text-center">{inner}</div>
  );
}

export function ErrorState({
  message,
  onRetry,
}: {
  message: string;
  onRetry?: () => void;
}) {
  return (
    <div
      className="card bg-[var(--red-bg)] px-[var(--space-card)] py-3 text-[13px] text-[var(--red)]"
      role="alert"
    >
      <p>{message}</p>
      {onRetry ? (
        <button type="button" className="btn mt-2" onClick={onRetry}>
          Yeniden dene
        </button>
      ) : null}
    </div>
  );
}

export function Field({
  id,
  label,
  error,
  children,
}: {
  id: string;
  label: string;
  error?: string;
  children: ReactNode;
}) {
  return (
    <div className="flex flex-col gap-1">
      <label htmlFor={id} className="meta-text">
        {label}
      </label>
      {children}
      {error ? (
        <p className="text-[12px] text-[var(--red)]" role="alert">
          {error}
        </p>
      ) : null}
    </div>
  );
}
