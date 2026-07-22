import { apiClient } from "@/lib/api-client";
import type { Harness } from "../types";

export function fetchHarnesses(): Promise<Harness[]> {
  return apiClient.get<Harness[]>("/harnesses");
}
