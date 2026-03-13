import { apiClient } from "@/lib/api-client";
import type { DirEntry } from "@/features/sessions";

interface ListDirResponse {
  path: string;
  entries: DirEntry[];
}

export async function listDirectory(path?: string, showHidden?: boolean): Promise<ListDirResponse> {
  const params: Record<string, string> = {};
  if (path) params.path = path;
  if (showHidden) params.showHidden = "true";
  return apiClient.get<ListDirResponse>("/fs", params);
}
