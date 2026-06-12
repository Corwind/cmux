import { createPortal } from "react-dom";
import { useToastStore, type Toast, type ToastVariant } from "./toast.store";

const VARIANT_STYLES: Record<
  ToastVariant,
  { bar: string; icon: React.ReactNode }
> = {
  error: {
    bar: "#f87171",
    icon: (
      <svg className="h-4 w-4 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
        <path strokeLinecap="round" strokeLinejoin="round" d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126zM12 15.75h.007v.008H12v-.008z" />
      </svg>
    ),
  },
  warning: {
    bar: "#eab308",
    icon: (
      <svg className="h-4 w-4 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
        <path strokeLinecap="round" strokeLinejoin="round" d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126zM12 15.75h.007v.008H12v-.008z" />
      </svg>
    ),
  },
  info: {
    bar: "var(--cmux-accent)",
    icon: (
      <svg className="h-4 w-4 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
        <path strokeLinecap="round" strokeLinejoin="round" d="M11.25 11.25l.041-.02a.75.75 0 011.063.852l-.708 2.836a.75.75 0 001.063.853l.041-.021M21 12a9 9 0 11-18 0 9 9 0 0118 0zm-9-3.75h.008v.008H12V8.25z" />
      </svg>
    ),
  },
  success: {
    bar: "#22c55e",
    icon: (
      <svg className="h-4 w-4 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
        <path strokeLinecap="round" strokeLinejoin="round" d="M9 12.75L11.25 15 15 9.75M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
      </svg>
    ),
  },
};

function ToastItem({ t }: { t: Toast }) {
  const dismiss = useToastStore((s) => s.dismiss);
  const { bar, icon } = VARIANT_STYLES[t.variant];

  return (
    <div
      role="alert"
      className="flex w-80 items-start overflow-hidden rounded-md shadow-lg"
      style={{
        backgroundColor: "var(--cmux-surface)",
        border: "1px solid var(--cmux-border)",
        color: "var(--cmux-text)",
      }}
    >
      {/* colored left bar */}
      <div className="w-1 self-stretch shrink-0" style={{ backgroundColor: bar }} />

      <div className="flex min-w-0 flex-1 items-start gap-2.5 px-3 py-2.5">
        <span style={{ color: bar, marginTop: 1 }}>{icon}</span>
        <div className="min-w-0 flex-1">
          <p className="text-xs font-semibold leading-snug" style={{ color: "var(--cmux-text)" }}>
            {t.title}
          </p>
          {t.message && (
            <p className="mt-0.5 text-xs leading-snug" style={{ color: "var(--cmux-text-secondary)" }}>
              {t.message}
            </p>
          )}
        </div>
        <button
          type="button"
          onClick={() => dismiss(t.id)}
          className="shrink-0 rounded p-0.5 transition-colors"
          style={{ color: "var(--cmux-text-faint)" }}
          aria-label="Dismiss"
          onMouseEnter={(e) => { e.currentTarget.style.color = "var(--cmux-text)"; }}
          onMouseLeave={(e) => { e.currentTarget.style.color = "var(--cmux-text-faint)"; }}
        >
          <svg className="h-3 w-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>
    </div>
  );
}

export function ToastContainer() {
  const toasts = useToastStore((s) => s.toasts);

  return createPortal(
    <div
      aria-live="polite"
      className="pointer-events-none fixed bottom-4 left-4 z-50 flex flex-col gap-2"
    >
      {toasts.map((t) => (
        <div key={t.id} className="pointer-events-auto">
          <ToastItem t={t} />
        </div>
      ))}
    </div>,
    document.body,
  );
}
