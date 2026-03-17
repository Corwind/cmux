import { useMutation, useQueryClient } from "@tanstack/react-query";
import { restartSession } from "../services/sessions-api";
import { sessionKeys } from "./useSessions";

export function useRestartSession() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => restartSession(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: sessionKeys.all });
    },
  });
}
