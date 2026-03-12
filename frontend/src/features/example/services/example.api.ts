import { apiClient } from "@/lib/api-client";
import type { ApiResponse } from "@/types/api.types";
import type { CreateExampleInput, Example } from "../types/example.types";

export function fetchExamples(): Promise<ApiResponse<Example[]>> {
  return apiClient.get<ApiResponse<Example[]>>("/examples");
}

export function fetchExampleById(id: string): Promise<ApiResponse<Example>> {
  return apiClient.get<ApiResponse<Example>>(`/examples/${id}`);
}

export function createExample(
  input: CreateExampleInput,
): Promise<ApiResponse<Example>> {
  return apiClient.post<ApiResponse<Example>>("/examples", input);
}
