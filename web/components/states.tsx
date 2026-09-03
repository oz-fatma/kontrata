import type { ReactNode } from "react";

export function LoadingState({ label = "Yükleniyor" }: { label?: string }) {
  return (
    <p className="py-8 text-[13px] text-[var(--muted)]" role="status">
      {label}…
    </p>
  );
}

export function EmptyState({
  title,
  detail,
}: {
  title: string;
  detail?: string;
}) {
  return (
    <div className="py-10 text-center">
      <p className="text-[15px] font-medium">{title}</p>
      {detail ? (
        <p className="mt-1 text-[13px] text-[var(--muted)]">{detail}</p>
      ) : null}
    </div>
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
      className="rounded-card bg-[var(--red-bg)] px-3 py-3 text-[13px] text-[var(--red)]"
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
      <label htmlFor={id} className="text-[13px] text-[var(--muted)]">
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
