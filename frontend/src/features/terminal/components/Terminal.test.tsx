import { act, render } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { Terminal } from "./Terminal";

type DataListener = (data: string) => void;
type ResizeListener = (size: { rows: number; cols: number }) => void;
type LinkHandler = (event: MouseEvent, uri: string) => void;
type LinkOptions = {
  hover?: () => void;
  leave?: () => void;
};

const xtermState = vi.hoisted(() => ({
  instances: [] as Array<{
    dataListeners: Set<DataListener>;
    resizeListeners: Set<ResizeListener>;
    options: Record<string, unknown>;
    element?: HTMLElement;
    emitData(data: string): void;
  }>,
  linkifierMouseDown: vi.fn(),
  webLinksHandler: undefined as LinkHandler | undefined,
  webLinksOptions: undefined as LinkOptions | undefined,
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
    element?: HTMLElement;

    constructor(options?: Record<string, unknown>) {
      if (options) this.options = options;
      xtermState.instances.push(this);
    }

    loadAddon() {}
    open(container: HTMLElement) {
      this.element = document.createElement("div");
      const screen = document.createElement("div");
      screen.className = "xterm-screen";
      screen.addEventListener("mousedown", xtermState.linkifierMouseDown);
      this.element.appendChild(screen);
      container.appendChild(this.element);
    }
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
vi.mock("@xterm/addon-web-links", () => ({
  WebLinksAddon: class {
    constructor(handler: LinkHandler, options: LinkOptions) {
      xtermState.webLinksHandler = handler;
      xtermState.webLinksOptions = options;
    }
  },
}));
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
    xtermState.linkifierMouseDown.mockClear();
    xtermState.webLinksHandler = undefined;
    xtermState.webLinksOptions = undefined;
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

  it("does not forward a web link click to the terminal mouse protocol", () => {
    render(
      <Terminal sessionId="session-1" wsBaseUrl="ws://localhost/session-1" />,
    );

    const terminal = xtermState.instances[0]!;
    const screen =
      terminal.element!.querySelector<HTMLElement>(".xterm-screen")!;
    const terminalMouseDown = vi.fn();
    terminal.element!.addEventListener("mousedown", terminalMouseDown);

    xtermState.webLinksOptions!.hover?.();
    screen.dispatchEvent(
      new MouseEvent("mousedown", { bubbles: true, button: 0 }),
    );

    expect(xtermState.linkifierMouseDown).toHaveBeenCalledOnce();
    expect(terminalMouseDown).not.toHaveBeenCalled();
  });

  it("does not forward an OSC hyperlink click to the terminal mouse protocol", () => {
    render(
      <Terminal sessionId="session-1" wsBaseUrl="ws://localhost/session-1" />,
    );

    const terminal = xtermState.instances[0]!;
    const screen =
      terminal.element!.querySelector<HTMLElement>(".xterm-screen")!;
    const terminalMouseDown = vi.fn();
    terminal.element!.addEventListener("mousedown", terminalMouseDown);
    const linkHandler = terminal.options.linkHandler as LinkOptions & {
      activate: LinkHandler;
    };

    linkHandler.hover?.();
    screen.dispatchEvent(
      new MouseEvent("mousedown", { bubbles: true, button: 0 }),
    );

    expect(xtermState.linkifierMouseDown).toHaveBeenCalledOnce();
    expect(terminalMouseDown).not.toHaveBeenCalled();
  });

  it("continues forwarding ordinary clicks to the terminal", () => {
    render(
      <Terminal sessionId="session-1" wsBaseUrl="ws://localhost/session-1" />,
    );

    const terminal = xtermState.instances[0]!;
    const screen =
      terminal.element!.querySelector<HTMLElement>(".xterm-screen")!;
    const terminalMouseDown = vi.fn();
    terminal.element!.addEventListener("mousedown", terminalMouseDown);

    screen.dispatchEvent(
      new MouseEvent("mousedown", { bubbles: true, button: 0 }),
    );

    expect(terminalMouseDown).toHaveBeenCalledOnce();
  });
});
