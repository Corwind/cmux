import { useQuery, useMutation } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import { toast } from "@/components/ui/Toast";
import type { WorktreeEntry } from "../types";

export const worktreeKeys = {
  all: ["worktrees"] as const,
};

export function fetchWorktrees(): Promise<WorktreeEntry[]> {
  return apiClient.get<WorktreeEntry[]>("/worktrees");
}

export function useWorktrees() {
  return useQuery({
    queryKey: worktreeKeys.all,
    queryFn: fetchWorktrees,
    refetchInterval: 5000,
  });
}

export function useDeleteWorktree() {
  return useMutation({
    mutationFn: (id: string) => apiClient.delete<void>(`/worktrees/${id}`),
    // No onSuccess invalidation — the worktree_deleted WS event handles that.
    onError: (err: unknown) => {
      const message = err instanceof Error ? err.message : "Failed to delete worktree.";
      toast("error", "Cannot delete worktree", message);
    },
  });
}

export function groupByRepo(entries: WorktreeEntry[]): Map<string, WorktreeEntry[]> {
  const map = new Map<string, WorktreeEntry[]>();
  for (const entry of entries) {
    const group = map.get(entry.repo_root) ?? [];
    group.push(entry);
    map.set(entry.repo_root, group);
  }
  return map;
}
