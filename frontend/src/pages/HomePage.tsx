import { useState } from "react";
import { AppLayout } from "@/components/layout/AppLayout";
import { SessionList, CreateSessionDialog, useSessionsStore } from "@/features/sessions";
import { TemplateManager } from "@/features/templates";
import { Terminal } from "@/features/terminal";

export function HomePage() {
  const activeSessionId = useSessionsStore((s) => s.activeSessionId);
  const [templatesOpen, setTemplatesOpen] = useState(false);

  return (
    <AppLayout
      sidebar={
        <>
          <CreateSessionDialog />
          <div className="mt-3">
            <SessionList />
          </div>
          <div className="mt-4 border-t border-gray-700 pt-3">
            <button
              type="button"
              onClick={() => setTemplatesOpen(!templatesOpen)}
              className="flex w-full items-center justify-between px-1 text-xs font-medium uppercase tracking-wider text-gray-500 hover:text-gray-300"
            >
              Templates
              <svg
                className={`h-3.5 w-3.5 transition-transform ${templatesOpen ? "rotate-180" : ""}`}
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
                strokeWidth={2}
              >
                <path strokeLinecap="round" strokeLinejoin="round" d="M19 9l-7 7-7-7" />
              </svg>
            </button>
            {templatesOpen && (
              <div className="mt-2">
                <TemplateManager />
              </div>
            )}
          </div>
        </>
      }
    >
      {activeSessionId ? (
        <Terminal key={activeSessionId} sessionId={activeSessionId} />
      ) : (
        <div className="flex h-full items-center justify-center">
          <div className="text-center text-gray-500">
            <div className="mb-2 font-mono text-4xl text-gray-700">&gt;_</div>
            <p className="text-sm">
              Select or create a session to start a terminal.
            </p>
          </div>
        </div>
      )}
    </AppLayout>
  );
}
