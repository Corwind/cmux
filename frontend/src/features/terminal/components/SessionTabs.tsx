import { cn } from "@/lib/cn";
import { useSessionsStore } from "@/features/sessions";

export function SessionTabs() {
  const { openTabs, activeSessionId, setActiveSession, closeTab } =
    useSessionsStore();

  if (openTabs.length === 0) {
    return null;
  }

  return (
    <div className="flex min-h-[36px] items-center gap-px overflow-x-auto bg-gray-900 px-1">
      {openTabs.map((tab) => (
        <div
          key={tab.id}
          className={cn(
            "group flex items-center gap-1.5 rounded-t px-3 py-1.5 text-xs font-medium transition-colors",
            activeSessionId === tab.id
              ? "bg-[#1a1b26] text-white"
              : "bg-gray-800 text-gray-400 hover:bg-gray-750 hover:text-gray-200",
          )}
        >
          <button
            type="button"
            onClick={() => setActiveSession(tab.id)}
            className="max-w-[120px] truncate"
          >
            {tab.name}
          </button>
          <button
            type="button"
            onClick={() => closeTab(tab.id)}
            className="rounded p-0.5 opacity-0 transition-opacity hover:bg-gray-700 group-hover:opacity-100"
            title="Close tab"
          >
            <svg
              className="h-3 w-3"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
              strokeWidth={2}
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                d="M6 18L18 6M6 6l12 12"
              />
            </svg>
          </button>
        </div>
      ))}
    </div>
  );
}
