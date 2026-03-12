import { useMutation, useQueryClient } from "@tanstack/react-query";
import { deleteSession } from "../services/sessions-api";
import { useSessionsStore } from "../stores/sessions.store";
import { sessionKeys } from "./useSessions";

export function useDeleteSession() {
  const queryClient = useQueryClient();
  const activeSessionId = useSessionsStore((s) => s.activeSessionId);
  const setActiveSession = useSessionsStore((s) => s.setActiveSession);

  return useMutation({
    mutationFn: (id: string) => deleteSession(id),
    onSuccess: (_data, deletedId) => {
      if (deletedId === activeSessionId) {
        setActiveSession(null);
      }
      void queryClient.invalidateQueries({ queryKey: sessionKeys.all });
    },
  });
}
