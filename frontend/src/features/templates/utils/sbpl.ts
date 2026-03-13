export interface SandboxRule {
  path: string;
  read: boolean;
  write: boolean;
  metadata: boolean;
}

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

const ALLOW_PATTERN =
  /^\(allow\s+(file-read\*|file-write\*|file-read-metadata)\s+\(subpath\s+"([^"]+)"\)\)$/;

export function sbplToRules(content: string): SandboxRule[] {
  const pathMap = new Map<
    string,
    { read: boolean; write: boolean; metadata: boolean }
  >();

  for (const line of content.split("\n")) {
    const trimmed = line.trim();
    if (!trimmed) continue;

    const match = ALLOW_PATTERN.exec(trimmed);
    if (!match) continue;

    const [, operation, path] = match;
    if (!pathMap.has(path)) {
      pathMap.set(path, { read: false, write: false, metadata: false });
    }
    const perms = pathMap.get(path)!;

    if (operation === "file-read*") perms.read = true;
    else if (operation === "file-write*") perms.write = true;
    else if (operation === "file-read-metadata") perms.metadata = true;
  }

  return Array.from(pathMap.entries()).map(([path, perms]) => ({
    path,
    ...perms,
  }));
}
