import { useMutation, useQueryClient } from "@tanstack/react-query";
import { createSession } from "../services/sessions-api";
import type { CreateSessionInput } from "../types";
import { sessionKeys } from "./useSessions";
import { useSessionsStore } from "../stores/sessions.store";

export function useCreateSession() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (input: CreateSessionInput) => createSession(input),
    onSuccess: (data) => {
      void queryClient.invalidateQueries({ queryKey: sessionKeys.all });
      useSessionsStore.getState().setActiveSession(data.id);
    },
  });
}
