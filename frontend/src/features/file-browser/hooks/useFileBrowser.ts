import { useQuery } from "@tanstack/react-query";
import { listDirectory } from "../services/filesystem-api";

export function useFileBrowser(path: string) {
  return useQuery({
    queryKey: ["filesystem", path],
    queryFn: () => listDirectory(path),
  });
}
