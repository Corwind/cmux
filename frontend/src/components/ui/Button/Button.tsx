import type { ButtonHTMLAttributes } from "react";
import { cn } from "@/lib/cn";

const variantClasses = {
  primary: "bg-primary text-white hover:bg-primary-dark",
  secondary: "bg-gray-200 text-text hover:bg-gray-300",
  outline:
    "border border-primary text-primary hover:bg-primary hover:text-white",
  ghost: "text-text hover:bg-gray-100",
} as const;

const sizeClasses = {
  sm: "px-3 py-1.5 text-sm",
  md: "px-4 py-2 text-base",
  lg: "px-6 py-3 text-lg",
} as const;

interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: "primary" | "secondary" | "outline" | "ghost";
  size?: "sm" | "md" | "lg";
  loading?: boolean;
}

export function Button({
  variant = "primary",
  size = "md",
  loading = false,
  disabled,
  className,
  children,
  ...props
}: ButtonProps) {
  return (
    <button
      className={cn(
        "inline-flex items-center justify-center rounded-md font-medium transition-colors focus:outline-none focus:ring-2 focus:ring-primary focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50",
        variantClasses[variant],
        sizeClasses[size],
        className,
      )}
      disabled={disabled || loading}
      {...props}
    >
      {loading && (
        <div
          className="mr-2 h-4 w-4 animate-spin rounded-full border-2 border-current border-t-transparent"
          role="status"
          aria-label="Loading"
        />
      )}
      {children}
    </button>
  );
}
