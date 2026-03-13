import type { SandboxRule } from "../types";

export function rulesToSbpl(rules: SandboxRule[]): string {
  const lines: string[] = [];

  for (const rule of rules) {
    if (!rule.path) continue;
    if (rule.read) {
      lines.push(`(allow file-read* (subpath "${rule.path}"))`);
    }
    if (rule.write) {
      lines.push(`(allow file-write* (subpath "${rule.path}"))`);
    }
    if (rule.metadata) {
      lines.push(`(allow file-read-metadata (subpath "${rule.path}"))`);
    }
  }

  return lines.join("\n");
}
