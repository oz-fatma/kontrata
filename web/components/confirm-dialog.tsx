"use client";

export function ConfirmDialog({
  title,
  message,
  confirmLabel = "Sil",
  cancelLabel = "Vazgeç",
  busy = false,
  onConfirm,
  onCancel,
}: {
  title: string;
  message: string;
  confirmLabel?: string;
  cancelLabel?: string;
  busy?: boolean;
  onConfirm: () => void;
  onCancel: () => void;
}) {
  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/30 px-4"
      role="presentation"
      onClick={onCancel}
    >
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby="onay-baslik"
        aria-describedby="onay-metin"
        className="w-full max-w-sm rounded-card border-[0.5px] border-[var(--line)] bg-[var(--bg)] p-4"
        onClick={(e) => e.stopPropagation()}
      >
        <h2 id="onay-baslik" className="mb-2">
          {title}
        </h2>
        <p id="onay-metin" className="mb-4 text-[13px] text-[var(--muted)]">
          {message}
        </p>
        <div className="flex justify-end gap-2">
          <button type="button" className="btn" disabled={busy} onClick={onCancel}>
            {cancelLabel}
          </button>
          <button
            type="button"
            className="btn btn-danger"
            disabled={busy}
            onClick={onConfirm}
          >
            {confirmLabel}
          </button>
        </div>
      </div>
    </div>
  );
}
