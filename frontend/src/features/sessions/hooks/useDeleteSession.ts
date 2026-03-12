import { useMutation, useQueryClient } from "@tanstack/react-query";
import { deleteSession } from "../services/sessions-api";
import { sessionKeys } from "./useSessions";

export function useDeleteSession() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => deleteSession(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: sessionKeys.all });
    },
  });
}
