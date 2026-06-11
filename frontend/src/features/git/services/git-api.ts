import { apiClient } from "@/lib/api-client";
import type { GitInfo } from "../types";

export function fetchGitInfo(path: string): Promise<GitInfo> {
  return apiClient.get<GitInfo>("/git/info", { path });
}
