import { useQuery } from "@tanstack/react-query";
import { listDirectory } from "../services/filesystem-api";

export function useFileBrowser(path?: string, showHidden?: boolean) {
  return useQuery({
    queryKey: ["filesystem", path ?? "home", showHidden ? "hidden" : "visible"],
    queryFn: () => listDirectory(path, showHidden),
  });
}
