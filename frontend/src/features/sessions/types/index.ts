export interface Session {
  id: string;
  name: string;
  working_dir: string;
  status: "running" | "stopped";
  pid: number;
  template_id: string;
  created_at: string;
  updated_at: string;
}

export interface CreateSessionInput {
  name?: string;
  working_dir: string;
  template_id?: string;
  skip_permissions?: boolean;
}

export interface DirEntry {
  name: string;
  is_dir: boolean;
}
