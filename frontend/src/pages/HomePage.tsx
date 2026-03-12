import { AppLayout } from "@/components/layout/AppLayout";
import { SessionList, CreateSessionDialog, useSessionsStore } from "@/features/sessions";
import { Terminal } from "@/features/terminal";

export function HomePage() {
  const activeSessionId = useSessionsStore((s) => s.activeSessionId);

  return (
    <AppLayout
      sidebar={
        <>
          <CreateSessionDialog />
          <div className="mt-3">
            <SessionList />
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
