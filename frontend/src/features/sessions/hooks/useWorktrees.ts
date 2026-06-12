import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
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
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => apiClient.delete<void>(`/worktrees/${id}`),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: worktreeKeys.all });
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
