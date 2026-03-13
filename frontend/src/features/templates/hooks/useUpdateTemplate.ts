import { useMutation, useQueryClient } from "@tanstack/react-query";
import { updateTemplate } from "../services/templates-api";
import type { UpdateTemplateInput } from "../types";
import { templateKeys } from "./useTemplates";

export function useUpdateTemplate() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: UpdateTemplateInput }) =>
      updateTemplate(id, input),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: templateKeys.all });
    },
  });
}
