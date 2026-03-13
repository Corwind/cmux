export interface SandboxRule {
  path: string;
  read: boolean;
  write: boolean;
  metadata: boolean;
}

export interface SandboxTemplate {
  id: string;
  name: string;
  content: string;
  rules: SandboxRule[];
  is_default: boolean;
  created_at: string;
  updated_at: string;
}

export interface CreateTemplateInput {
  name: string;
  rules: SandboxRule[];
}

export interface UpdateTemplateInput {
  name: string;
  rules: SandboxRule[];
}
