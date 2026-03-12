import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  createExample,
  fetchExampleById,
  fetchExamples,
} from "../services/example.api";
import type { CreateExampleInput } from "../types/example.types";

export const exampleKeys = {
  all: ["examples"] as const,
  detail: (id: string) => ["examples", id] as const,
};

export function useExamples() {
  return useQuery({
    queryKey: exampleKeys.all,
    queryFn: fetchExamples,
  });
}

export function useExample(id: string) {
  return useQuery({
    queryKey: exampleKeys.detail(id),
    queryFn: () => fetchExampleById(id),
  });
}

export function useCreateExample() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (input: CreateExampleInput) => createExample(input),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: exampleKeys.all });
    },
  });
}
