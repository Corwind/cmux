import { cn } from "@/lib/cn";
import type { Session } from "../types";

const statusStyles: Record<Session["status"], string> = {
  running: "bg-green-500/20 text-green-400",
  stopped: "bg-gray-500/20 text-gray-400",
};

interface StatusBadgeProps {
  status: Session["status"];
}

export function StatusBadge({ status }: StatusBadgeProps) {
  return (
    <span
      className={cn(
        "inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium",
        statusStyles[status],
      )}
    >
      <span
        className={cn("mr-1.5 h-1.5 w-1.5 rounded-full", {
          "bg-green-400": status === "running",
          "bg-gray-400": status === "stopped",
        })}
      />
      {status}
    </span>
  );
}
