interface LoadingSpinnerProps {
  size?: "sm" | "md" | "lg";
}

const sizeClasses = {
  sm: "h-5 w-5 border-2",
  md: "h-8 w-8 border-3",
  lg: "h-12 w-12 border-4",
} as const;

export function LoadingSpinner({ size = "md" }: LoadingSpinnerProps) {
  return (
    <div className="flex items-center justify-center p-4">
      <div
        className={`animate-spin rounded-full border-primary border-t-transparent ${sizeClasses[size]}`}
        role="status"
        aria-label="Loading"
      />
    </div>
  );
}
