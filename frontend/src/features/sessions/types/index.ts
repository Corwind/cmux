export interface Session {
  id: string;
  name: string;
  working_dir: string;
  status: "running" | "stopped" | "provisioning" | "failed";
  pid: number;
  template_id: string;
  skip_permissions: boolean;
  repo_root?: string;
  git_branch?: string;
  worktree_managed?: boolean;
  error?: string;
  created_at: string;
  updated_at: string;
}

export interface WorktreeInput {
  repo_path: string;
  branch: string;
  base_ref?: string;
  create_branch: boolean;
  path?: string;
}

export interface CreateSessionInput {
  name?: string;
  working_dir: string;
  template_id?: string;
  skip_permissions?: boolean;
  worktree?: WorktreeInput;
}

export interface DirEntry {
  name: string;
  is_dir: boolean;
}

export interface WorktreeEntry {
  id: string;
  path: string;
  branch: string;
  repo_root: string;
  created_at: string;
  session_id?: string;
  session_name?: string;
  session_status?: "running" | "stopped" | "provisioning" | "failed";
}
