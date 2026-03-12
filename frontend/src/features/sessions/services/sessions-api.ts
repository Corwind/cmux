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

export function deleteSession(id: string): Promise<void> {
  return apiClient.delete<void>(`/sessions/${id}`);
}
