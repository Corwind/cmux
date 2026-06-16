import { cn } from "@/lib/cn";
import type { Session } from "../types";

interface StatusBadgeProps {
  status: Session["status"];
  hasNotification?: boolean;
}

export function StatusBadge({ status, hasNotification }: StatusBadgeProps) {
  return (
    <span
      className={cn("inline-block h-2 w-2 shrink-0 rounded-full", {
        "animate-pulse bg-yellow-400": hasNotification,
        "bg-green-400": !hasNotification && status === "running",
        "bg-gray-500": !hasNotification && status === "stopped",
      })}
      aria-label={hasNotification ? "needs attention" : status}
    />
  );
}
