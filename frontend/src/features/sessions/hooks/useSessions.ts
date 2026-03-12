import { useQuery } from "@tanstack/react-query";
import { fetchSessions } from "../services/sessions-api";

export const sessionKeys = {
  all: ["sessions"] as const,
  detail: (id: string) => ["sessions", id] as const,
};

export function useSessions() {
  return useQuery({
    queryKey: sessionKeys.all,
    queryFn: fetchSessions,
    refetchInterval: 5000,
  });
}
