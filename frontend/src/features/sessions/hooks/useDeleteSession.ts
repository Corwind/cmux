import { useMutation, useQueryClient } from "@tanstack/react-query";
import { ApiError } from "@/lib/api-client";
import { deleteSession, type WorktreeDeleteAction } from "../services/sessions-api";
import { useSessionsStore } from "../stores/sessions.store";
import { sessionKeys } from "./useSessions";
import type { Session } from "../types";

interface DeleteSessionArgs {
  id: string;
  worktreeAction?: WorktreeDeleteAction;
}

export function useDeleteSession() {
  const queryClient = useQueryClient();
  const activeSessionId = useSessionsStore((s) => s.activeSessionId);
  const setActiveSession = useSessionsStore((s) => s.setActiveSession);

  return useMutation({
    mutationFn: ({ id, worktreeAction }: DeleteSessionArgs) =>
      deleteSession(id, worktreeAction),
    onSuccess: (_data, { id: deletedId }) => {
      if (deletedId === activeSessionId) {
        setActiveSession(null);
      }
      void queryClient.invalidateQueries({ queryKey: sessionKeys.all });
    },
  });
}

export async function deleteSessionWithWorktreePrompt(
  session: Session,
  deleteMutation: ReturnType<typeof useDeleteSession>,
): Promise<void> {
  if (!session.worktree_managed) {
    return new Promise((resolve, reject) => {
      deleteMutation.mutate(
        { id: session.id },
        { onSuccess: () => resolve(), onError: (e) => reject(e) },
      );
    });
  }

  const removeWorktree = window.confirm(
    `Session "${session.name}" uses a managed worktree at:\n${session.working_dir}\n\nAlso remove the worktree?`,
  );

  if (!removeWorktree) {
    return new Promise((resolve, reject) => {
      deleteMutation.mutate(
        { id: session.id, worktreeAction: "keep" },
        { onSuccess: () => resolve(), onError: (e) => reject(e) },
      );
    });
  }

  return new Promise((resolve, reject) => {
    deleteMutation.mutate(
      { id: session.id, worktreeAction: "remove" },
      {
        onSuccess: () => resolve(),
        onError: (err) => {
          if (err instanceof ApiError && err.status === 409) {
            const force = window.confirm(
              `The worktree has uncommitted changes.\n\nForce remove anyway?`,
            );
            if (force) {
              deleteMutation.mutate(
                { id: session.id, worktreeAction: "force" },
                { onSuccess: () => resolve(), onError: (e) => reject(e) },
              );
            } else {
              deleteMutation.mutate(
                { id: session.id, worktreeAction: "keep" },
                { onSuccess: () => resolve(), onError: (e) => reject(e) },
              );
            }
          } else {
            reject(err);
          }
        },
      },
    );
  });
}
