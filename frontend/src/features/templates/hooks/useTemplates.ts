import { useQuery } from "@tanstack/react-query";
import { fetchTemplates } from "../services/templates-api";

export const templateKeys = {
  all: ["templates"] as const,
  detail: (id: string) => ["templates", id] as const,
};

export function useTemplates() {
  return useQuery({
    queryKey: templateKeys.all,
    queryFn: fetchTemplates,
    refetchInterval: 10000,
  });
}
