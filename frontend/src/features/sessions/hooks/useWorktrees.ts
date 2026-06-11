import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { deleteWorktree, fetchWorktrees } from "../services/worktrees-api";

export const worktreeKeys = {
  all: ["worktrees"] as const,
};

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
    mutationFn: ({ id, force }: { id: string; force?: boolean }) =>
      deleteWorktree(id, force),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: worktreeKeys.all });
    },
  });
}
