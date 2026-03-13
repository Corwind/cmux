export interface SandboxTemplate {
  id: string;
  name: string;
  content: string;
  is_default: boolean;
  created_at: string;
  updated_at: string;
}

export interface CreateTemplateInput {
  name: string;
  content: string;
}

export interface UpdateTemplateInput {
  name: string;
  content: string;
}
