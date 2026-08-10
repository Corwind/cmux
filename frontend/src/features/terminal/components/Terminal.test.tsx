import { act, render } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { Terminal } from "./Terminal";

type DataListener = (data: string) => void;
type ResizeListener = (size: { rows: number; cols: number }) => void;

const xtermState = vi.hoisted(() => ({
  instances: [] as Array<{
    dataListeners: Set<DataListener>;
    resizeListeners: Set<ResizeListener>;
    emitData(data: string): void;
  }>,
}));

vi.mock("@xterm/xterm", () => ({
  Terminal: class MockTerminal {
    rows = 24;
    cols = 80;
    options: Record<string, unknown> = {};
    unicode = { activeVersion: "" };
    parser = { registerCsiHandler: vi.fn(() => ({ dispose: vi.fn() })) };
    dataListeners = new Set<DataListener>();
    resizeListeners = new Set<ResizeListener>();

    constructor() {
      xtermState.instances.push(this);
    }

    loadAddon() {}
    open() {}
    write() {}
    dispose() {}

    onData(listener: DataListener) {
      this.dataListeners.add(listener);
      return { dispose: () => this.dataListeners.delete(listener) };
    }

    onResize(listener: ResizeListener) {
      this.resizeListeners.add(listener);
      return { dispose: () => this.resizeListeners.delete(listener) };
    }

    emitData(data: string) {
      for (const listener of this.dataListeners) listener(data);
    }
  },
}));

vi.mock("@xterm/addon-fit", () => ({
  FitAddon: class MockFitAddon {
    fit() {}
  },
}));
vi.mock("@xterm/addon-web-links", () => ({ WebLinksAddon: class {} }));
vi.mock("@xterm/addon-unicode11", () => ({ Unicode11Addon: class {} }));
vi.mock("@xterm/addon-webgl", () => ({
  WebglAddon: class MockWebglAddon {
    onContextLoss() {}
    dispose() {}
  },
}));

class MockWebSocket {
  static readonly CONNECTING = 0;
  static readonly OPEN = 1;
  static readonly CLOSED = 3;
  static instances: MockWebSocket[] = [];

  readyState = MockWebSocket.CONNECTING;
  binaryType = "";
  onopen: (() => void) | null = null;
  onmessage: ((event: MessageEvent) => void) | null = null;
  onclose: (() => void) | null = null;
  send = vi.fn();

  constructor(readonly url: string) {
    MockWebSocket.instances.push(this);
  }

  open() {
    this.readyState = MockWebSocket.OPEN;
    this.onopen?.();
  }

  serverClose() {
    this.readyState = MockWebSocket.CLOSED;
    this.onclose?.();
  }

  close() {
    this.readyState = MockWebSocket.CLOSED;
  }
}

class MockResizeObserver {
  observe() {}
  disconnect() {}
}

describe("Terminal WebSocket lifecycle", () => {
  let widthSpy: ReturnType<typeof vi.spyOn>;
  let heightSpy: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    vi.useFakeTimers();
    xtermState.instances.length = 0;
    MockWebSocket.instances.length = 0;
    vi.stubGlobal("WebSocket", MockWebSocket);
    vi.stubGlobal("ResizeObserver", MockResizeObserver);
    widthSpy = vi
      .spyOn(HTMLElement.prototype, "clientWidth", "get")
      .mockReturnValue(800);
    heightSpy = vi
      .spyOn(HTMLElement.prototype, "clientHeight", "get")
      .mockReturnValue(600);
  });

  afterEach(() => {
    widthSpy.mockRestore();
    heightSpy.mockRestore();
    vi.unstubAllGlobals();
    vi.useRealTimers();
  });

  it("keeps one input and resize subscription across reconnects", () => {
    const { unmount } = render(
      <Terminal sessionId="session-1" wsBaseUrl="ws://localhost/session-1" />,
    );

    const terminal = xtermState.instances[0]!;
    const firstSocket = MockWebSocket.instances[0]!;
    expect(terminal.dataListeners).toHaveLength(1);
    expect(terminal.resizeListeners).toHaveLength(1);

    act(() => firstSocket.serverClose());
    act(() => vi.advanceTimersByTime(1000));

    const secondSocket = MockWebSocket.instances[1]!;
    expect(MockWebSocket.instances).toHaveLength(2);
    expect(terminal.dataListeners).toHaveLength(1);
    expect(terminal.resizeListeners).toHaveLength(1);

    act(() => secondSocket.open());
    act(() => terminal.emitData("x"));
    expect(secondSocket.send).toHaveBeenCalledTimes(2); // initial resize + input
    expect(firstSocket.send).not.toHaveBeenCalled();

    unmount();
    expect(terminal.dataListeners).toHaveLength(0);
    expect(terminal.resizeListeners).toHaveLength(0);
  });
});
