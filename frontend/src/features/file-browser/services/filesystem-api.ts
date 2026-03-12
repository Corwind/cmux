import { apiClient } from "@/lib/api-client";
import type { DirEntry } from "@/features/sessions";

interface ListDirResponse {
  path: string;
  entries: DirEntry[];
}

export async function listDirectory(path: string): Promise<DirEntry[]> {
  const response = await apiClient.get<ListDirResponse>("/fs", { path });
  return response.entries ?? [];
}
