import { env } from "./env";

export const featureFlags = {
  enableMockApi: env.isDev,
  enableDevTools: env.isDev,
} as const;

export type FeatureFlags = typeof featureFlags;
