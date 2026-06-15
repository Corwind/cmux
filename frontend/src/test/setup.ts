import "@testing-library/jest-dom/vitest";
import { cleanup } from "@testing-library/react";
import { afterAll, afterEach, beforeAll } from "vitest";
import { server } from "./mocks/server";

// Node.js v22+ exposes `localStorage` as an accessor property returning
// `undefined` unless --localstorage-file is provided. This shadows jsdom's
// implementation and breaks zustand's `persist` middleware. Override it with
// an in-memory Map-backed implementation so tests can use localStorage freely.
const _localStorageMap = new Map<string, string>();
const localStorageMock: Storage = {
  get length() {
    return _localStorageMap.size;
  },
  key(index: number) {
    return Array.from(_localStorageMap.keys())[index] ?? null;
  },
  getItem(key: string) {
    return _localStorageMap.get(key) ?? null;
  },
  setItem(key: string, value: string) {
    _localStorageMap.set(key, value);
  },
  removeItem(key: string) {
    _localStorageMap.delete(key);
  },
  clear() {
    _localStorageMap.clear();
  },
};
Object.defineProperty(globalThis, "localStorage", {
  value: localStorageMock,
  writable: true,
  configurable: true,
});

beforeAll(() => server.listen({ onUnhandledRequest: "bypass" }));
afterEach(() => {
  cleanup();
  server.resetHandlers();
});
afterAll(() => server.close());
