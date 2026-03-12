import type { ReactNode } from "react";

interface AppLayoutProps {
  sidebar: ReactNode;
  tabs: ReactNode;
  children: ReactNode;
}

export function AppLayout({ sidebar, tabs, children }: AppLayoutProps) {
  return (
    <div className="flex h-screen w-screen overflow-hidden bg-gray-950 text-white">
      <aside className="flex w-[280px] shrink-0 flex-col border-r border-gray-800 bg-gray-900">
        <div className="flex items-center border-b border-gray-800 px-4 py-3">
          <h1 className="font-mono text-sm font-bold tracking-wider text-green-400">
            cmux
          </h1>
        </div>
        <div className="flex flex-1 flex-col overflow-y-auto p-3">
          {sidebar}
        </div>
      </aside>

      <div className="flex flex-1 flex-col overflow-hidden">
        {tabs}
        <div className="flex-1 overflow-hidden">{children}</div>
      </div>
    </div>
  );
}
