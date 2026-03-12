export interface Session {
  id: string;
  name: string;
  working_dir: string;
  status: "running" | "stopped";
  pid: number;
  created_at: string;
  updated_at: string;
}

export interface CreateSessionInput {
  name: string;
  working_dir: string;
}

export interface DirEntry {
  name: string;
  is_dir: boolean;
}
