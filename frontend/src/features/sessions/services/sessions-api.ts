import { apiClient } from "@/lib/api-client";
import type { CreateSessionInput, Session } from "../types";

export function fetchSessions(): Promise<Session[]> {
  return apiClient.get<Session[]>("/sessions");
}

export function fetchSession(id: string): Promise<Session> {
  return apiClient.get<Session>(`/sessions/${id}`);
}

export function createSession(input: CreateSessionInput): Promise<Session> {
  return apiClient.post<Session>("/sessions", input);
}

export function resumeSession(id: string): Promise<Session> {
  return apiClient.post<Session>(`/sessions/${id}/resume`);
}

export function restartSession(id: string): Promise<Session> {
  return apiClient.post<Session>(`/sessions/${id}/restart`);
}

export type WorktreeDeleteAction = "keep" | "remove" | "force";

export function deleteSession(id: string, worktreeAction?: WorktreeDeleteAction): Promise<void> {
  const params = worktreeAction && worktreeAction !== "keep"
    ? { worktree: worktreeAction }
    : undefined;
  return apiClient.delete<void>(`/sessions/${id}`, { params });
}
