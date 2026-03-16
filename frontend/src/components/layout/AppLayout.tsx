import { useState, useCallback, useRef, type ReactNode } from "react";
import { useThemeVars } from "@/features/terminal";
import { SettingsModal } from "./SettingsModal";

interface AppLayoutProps {
  sidebar: ReactNode;
  children: ReactNode;
}

export function AppLayout({ sidebar, children }: AppLayoutProps) {
  const [sidebarWidth, setSidebarWidth] = useState(280);
  const isDragging = useRef(false);
  const themeVars = useThemeVars();
  const [settingsOpen, setSettingsOpen] = useState(false);

  const handleMouseDown = useCallback((e: React.MouseEvent) => {
    e.preventDefault();
    isDragging.current = true;

    const handleMouseMove = (e: MouseEvent) => {
      if (!isDragging.current) return;
      const newWidth = Math.min(Math.max(e.clientX, 180), 600);
      setSidebarWidth(newWidth);
    };

    const handleMouseUp = () => {
      isDragging.current = false;
      document.removeEventListener("mousemove", handleMouseMove);
      document.removeEventListener("mouseup", handleMouseUp);
      document.body.style.cursor = "";
      document.body.style.userSelect = "";
    };

    document.body.style.cursor = "col-resize";
    document.body.style.userSelect = "none";
    document.addEventListener("mousemove", handleMouseMove);
    document.addEventListener("mouseup", handleMouseUp);
  }, []);

  return (
    <div
      className="flex h-screen w-screen overflow-hidden"
      style={{
        ...themeVars,
        backgroundColor: "var(--cmux-bg)",
        color: "var(--cmux-text)",
      }}
    >
      <aside
        className="flex shrink-0 flex-col"
        style={{
          width: sidebarWidth,
          backgroundColor: "var(--cmux-sidebar)",
          borderRight: "1px solid var(--cmux-border)",
        }}
      >
        <div
          className="flex items-center px-4 py-3"
          style={{ borderBottom: "1px solid var(--cmux-border)" }}
        >
          <h1
            className="font-mono text-sm font-bold tracking-wider"
            style={{ color: "var(--cmux-accent)" }}
          >
            cmux
          </h1>
        </div>
        <div className="flex flex-1 flex-col overflow-y-auto p-3">
          {sidebar}
        </div>
        <div
          className="flex items-center px-3 py-2.5"
          style={{ borderTop: "1px solid var(--cmux-border)" }}
        >
          <button
            type="button"
            onClick={() => setSettingsOpen(true)}
            className="rounded p-1.5 transition-colors"
            style={{ color: "var(--cmux-text-secondary)" }}
            title="Settings"
            onMouseEnter={(e) => {
              e.currentTarget.style.color = "var(--cmux-text)";
              e.currentTarget.style.backgroundColor = "var(--cmux-surface-hover)";
            }}
            onMouseLeave={(e) => {
              e.currentTarget.style.color = "var(--cmux-text-secondary)";
              e.currentTarget.style.backgroundColor = "";
            }}
          >
            <svg
              className="h-5 w-5"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
              strokeWidth={1.5}
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.066 2.573c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.573 1.066c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.066-2.573c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"
              />
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"
              />
            </svg>
          </button>
        </div>
      </aside>

      <div
        onMouseDown={handleMouseDown}
        className="w-1 shrink-0 cursor-col-resize transition-colors"
        style={{ backgroundColor: "var(--cmux-border)" }}
        onMouseEnter={(e) => {
          e.currentTarget.style.backgroundColor = "var(--cmux-accent)";
        }}
        onMouseLeave={(e) => {
          e.currentTarget.style.backgroundColor = "var(--cmux-border)";
        }}
      />

      <div className="flex min-w-0 flex-1 flex-col overflow-hidden">
        <div className="relative flex-1 overflow-hidden">{children}</div>
      </div>

      <SettingsModal
        open={settingsOpen}
        onClose={() => setSettingsOpen(false)}
      />
    </div>
  );
}
