import { Link, Outlet } from "react-router";
import { ErrorBoundary } from "@/components/common";
import { env } from "@/config/env";

export function RootLayout() {
  return (
    <div className="flex min-h-screen flex-col bg-background font-sans text-text">
      <header className="sticky top-0 z-10 border-b border-gray-200 bg-white/80 backdrop-blur-sm">
        <nav className="mx-auto flex max-w-7xl items-center justify-between px-4 py-4 sm:px-6 lg:px-8">
          <Link
            to="/"
            className="text-xl font-bold tracking-tight text-primary"
          >
            {env.appTitle}
          </Link>
          <ul className="flex items-center gap-6">
            <li>
              <Link
                to="/"
                className="text-sm font-medium text-text-muted transition-colors hover:text-primary"
              >
                Home
              </Link>
            </li>
          </ul>
        </nav>
      </header>

      <main className="flex-1">
        <ErrorBoundary>
          <Outlet />
        </ErrorBoundary>
      </main>

      <footer className="border-t border-gray-200 bg-surface">
        <div className="mx-auto max-w-7xl px-4 py-6 text-center text-sm text-text-muted sm:px-6 lg:px-8">
          {env.appTitle} &mdash; Built with React, TypeScript & Tailwind CSS
        </div>
      </footer>
    </div>
  );
}
