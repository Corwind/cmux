import { apiClient } from "@/lib/api-client";
import type { WorktreeEntry } from "../types";

export function fetchWorktrees(): Promise<WorktreeEntry[]> {
  return apiClient.get<WorktreeEntry[]>("/worktrees");
}

export function deleteWorktree(id: string, force?: boolean): Promise<void> {
  const params = force ? { force: "true" } : undefined;
  return apiClient.delete<void>(`/worktrees/${id}`, { params });
}
