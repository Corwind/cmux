import { useQuery } from "@tanstack/react-query";
import { fetchGitInfo } from "../services/git-api";

export function useGitInfo(path: string | undefined) {
  return useQuery({
    queryKey: ["git", "info", path],
    queryFn: () => fetchGitInfo(path!),
    enabled: !!path,
    staleTime: 10_000,
  });
}
