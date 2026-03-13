import { apiClient } from "@/lib/api-client";
import type {
  CreateTemplateInput,
  SandboxTemplate,
  UpdateTemplateInput,
} from "../types";

export function fetchTemplates(): Promise<SandboxTemplate[]> {
  return apiClient.get<SandboxTemplate[]>("/templates");
}

export function fetchTemplate(id: string): Promise<SandboxTemplate> {
  return apiClient.get<SandboxTemplate>(`/templates/${id}`);
}

export function createTemplate(
  input: CreateTemplateInput,
): Promise<SandboxTemplate> {
  return apiClient.post<SandboxTemplate>("/templates", input);
}

export function updateTemplate(
  id: string,
  input: UpdateTemplateInput,
): Promise<SandboxTemplate> {
  return apiClient.put<SandboxTemplate>(`/templates/${id}`, input);
}

export function deleteTemplate(id: string): Promise<void> {
  return apiClient.delete<void>(`/templates/${id}`);
}

export function setDefaultTemplate(id: string): Promise<void> {
  return apiClient.post<void>(`/templates/${id}/default`);
}

export function clearDefaultTemplate(): Promise<void> {
  return apiClient.delete<void>("/templates/default");
}

export function importTemplate(
  input: CreateTemplateInput,
): Promise<SandboxTemplate> {
  return apiClient.post<SandboxTemplate>("/templates/import", input);
}

export async function exportTemplate(id: string): Promise<Blob> {
  const response = await fetch(
    `${window.location.origin}/api/templates/${id}/export`,
  );
  if (!response.ok) {
    throw new Error(`Export failed: ${response.statusText}`);
  }
  return response.blob();
}
