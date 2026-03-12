function getEnvVar(key: string, fallback?: string): string {
  const value = import.meta.env[key] ?? fallback;
  if (value === undefined) {
    throw new Error(`Missing required environment variable: ${key}`);
  }
  return value;
}

export const env = {
  apiBaseUrl: getEnvVar("VITE_API_BASE_URL", "http://localhost:3001/api"),
  appTitle: getEnvVar("VITE_APP_TITLE", "Cmux Frontend"),
  isDev: import.meta.env.DEV,
  isProd: import.meta.env.PROD,
  mode: import.meta.env.MODE,
} as const;
