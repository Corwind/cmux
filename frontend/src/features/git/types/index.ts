export interface GitWorktree {
  path: string;
  branch: string;
  head: string;
  is_main: boolean;
  detached: boolean;
  locked: boolean;
}

export interface GitBranch {
  name: string;
  is_current: boolean;
  is_remote: boolean;
}

export interface GitInfo {
  is_repo: boolean;
  repo_root?: string;
  current_branch?: string;
  worktrees?: GitWorktree[];
  branches?: GitBranch[];
}
