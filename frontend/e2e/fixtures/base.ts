import { test as base } from "@playwright/test";

// eslint-disable-next-line @typescript-eslint/no-empty-object-type
export const test = base.extend<{
  /* custom fixtures can go here */
}>({});
export { expect } from "@playwright/test";
