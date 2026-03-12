import { Link } from "react-router";
import { PageLayout } from "@/components/layout";

export function NotFoundPage() {
  return (
    <PageLayout title="Page Not Found">
      <div className="py-12 text-center">
        <p className="text-6xl font-bold text-primary">404</p>
        <p className="mt-4 text-lg text-text-muted">
          The page you are looking for does not exist or has been moved.
        </p>
        <Link
          to="/"
          className="mt-8 inline-block rounded-md bg-primary px-6 py-3 text-sm font-medium text-white transition-colors hover:bg-primary-dark"
        >
          Go back home
        </Link>
      </div>
    </PageLayout>
  );
}
