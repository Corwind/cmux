import { useMutation, useQueryClient } from "@tanstack/react-query";
import { createSession } from "../services/sessions-api";
import type { CreateSessionInput } from "../types";
import { sessionKeys } from "./useSessions";

export function useCreateSession() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (input: CreateSessionInput) => createSession(input),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: sessionKeys.all });
    },
  });
}
