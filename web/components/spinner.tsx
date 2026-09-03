export function Spinner({ label = "İşleniyor" }: { label?: string }) {
  return (
    <span className="spinner" role="status" aria-label={label} />
  );
}
