export interface Session {
  id: string;
  name: string;
  working_dir: string;
  status: "running" | "stopped" | "provisioning" | "failed";
  pid: number;
  template_id: string;
  skip_permissions: boolean;
  harness_type: string;
  repo_root?: string;
  git_branch?: string;
  worktree_managed?: boolean;
  error_message?: string;
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
  harness_type?: string;
  worktree?: WorktreeInput;
}

export interface DirEntry {
  name: string;
  is_dir: boolean;
  is_symlink?: boolean;
  is_broken?: boolean;
}

export interface WorktreeEntry {
  id: string;
  path: string;
  branch: string;
  repo_root: string;
  status: "ready" | "deleting";
  created_at: string;
  session_id?: string;
  session_name?: string;
  session_status?: "running" | "stopped" | "provisioning" | "failed";
}
