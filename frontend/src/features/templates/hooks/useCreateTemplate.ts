import { useMutation, useQueryClient } from "@tanstack/react-query";
import { createTemplate } from "../services/templates-api";
import type { CreateTemplateInput } from "../types";
import { templateKeys } from "./useTemplates";

export function useCreateTemplate() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (input: CreateTemplateInput) => createTemplate(input),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: templateKeys.all });
    },
  });
}
