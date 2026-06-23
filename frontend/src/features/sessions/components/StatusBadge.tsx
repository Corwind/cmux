import { cn } from "@/lib/cn";
import type { Session } from "../types";

interface StatusBadgeProps {
  status: Session["status"];
  hasNotification?: boolean;
}

export function StatusBadge({ status, hasNotification }: StatusBadgeProps) {
  function ariaLabel() {
    if (hasNotification) return "needs attention";
    if (status === "provisioning") return "creating worktree";
    return status;
  }

  return (
    <span
      className={cn("inline-block h-2 w-2 shrink-0 rounded-full", {
        "animate-pulse bg-yellow-400": hasNotification,
        "bg-green-400": !hasNotification && status === "running",
        "bg-gray-500": !hasNotification && status === "stopped",
        "animate-pulse bg-blue-400":
          !hasNotification && status === "provisioning",
        "bg-red-500": !hasNotification && status === "failed",
      })}
      aria-label={ariaLabel()}
    />
  );
}
