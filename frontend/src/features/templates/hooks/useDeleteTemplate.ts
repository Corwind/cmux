import { useMutation, useQueryClient } from "@tanstack/react-query";
import { deleteTemplate } from "../services/templates-api";
import { templateKeys } from "./useTemplates";

export function useDeleteTemplate() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => deleteTemplate(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: templateKeys.all });
    },
  });
}
