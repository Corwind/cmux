import type { ReactNode } from "react";

interface PageLayoutProps {
  title?: string;
  children: ReactNode;
}

export function PageLayout({ title, children }: PageLayoutProps) {
  return (
    <section className="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
      {title && (
        <h1 className="mb-8 text-3xl font-bold tracking-tight text-text">
          {title}
        </h1>
      )}
      {children}
    </section>
  );
}
