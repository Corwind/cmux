import { useEffect, useRef } from "react";
import { Terminal as XTerm } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import { WebLinksAddon } from "@xterm/addon-web-links";
import { Unicode11Addon } from "@xterm/addon-unicode11";
import { getTerminalTheme } from "../themes";
import { useTerminalThemeStore } from "../stores/terminal-theme.store";
import "@xterm/xterm/css/xterm.css";

interface TerminalProps {
  sessionId: string;
  wsBaseUrl?: string;
}

export function Terminal({ sessionId, wsBaseUrl }: TerminalProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const termRef = useRef<XTerm | null>(null);
  const cleanupRef = useRef<(() => void) | null>(null);
  const themeId = useTerminalThemeStore((s) => s.themeId);

  // Apply theme changes to a running terminal without remounting
  useEffect(() => {
    if (termRef.current) {
      const { theme } = getTerminalTheme(themeId);
      termRef.current.options.theme = theme;
    }
  }, [themeId]);

  useEffect(() => {
    // Clean up previous instance immediately
    cleanupRef.current?.();

    const container = containerRef.current;
    if (!container) return;

    let term: XTerm | null = null;
    let ws: WebSocket | null = null;
    let resizeObserver: ResizeObserver | null = null;
    let resizeTimer: ReturnType<typeof setTimeout>;
    let reconnectTimer: ReturnType<typeof setTimeout>;
    let alive = true;
    let intentionalClose = false;

    // Workaround: WKWebView (Tauri/macOS) fires a duplicate input event after
    // compositionend for dead keys (e.g. single quote, backtick), causing xterm.js
    // to emit the character twice via onData. We track compositionend and suppress
    // the second identical emission.
    let compositionEndData: string | null = null;
    let compositionFirstEmitted = false;
    let compositionTimer: ReturnType<typeof setTimeout>;

    const wsUrl =
      wsBaseUrl ?? `ws://${window.location.host}/ws/sessions/${sessionId}`;

    function connectWs(currentTerm: XTerm, fitAddon: FitAddon, encoder: TextEncoder) {
      if (!alive) return;

      ws = new WebSocket(wsUrl);
      ws.binaryType = "arraybuffer";
      const currentWs = ws;

      currentWs.onopen = () => {
        if (!alive) return;
        fitAddon.fit();
        currentWs.send(JSON.stringify({ type: "resize", rows: currentTerm.rows, cols: currentTerm.cols }));
      };

      currentWs.onmessage = (event: MessageEvent) => {
        if (!alive || !currentTerm) return;
        if (event.data instanceof ArrayBuffer) {
          currentTerm.write(new Uint8Array(event.data));
        } else if (event.data instanceof Blob) {
          event.data.arrayBuffer().then((buf) => {
            if (alive && currentTerm) currentTerm.write(new Uint8Array(buf));
          });
        } else {
          currentTerm.write(event.data);
        }
      };

      currentWs.onclose = () => {
        if (!alive || intentionalClose) return;
        reconnectTimer = setTimeout(() => connectWs(currentTerm, fitAddon, encoder), 1000);
      };

      currentTerm.onData((data) => {
        // Suppress duplicate dead-key emission from WKWebView (see compositionend handler)
        if (compositionEndData !== null && data === compositionEndData) {
          if (compositionFirstEmitted) {
            compositionEndData = null;
            compositionFirstEmitted = false;
            return;
          }
          compositionFirstEmitted = true;
        } else if (compositionEndData !== null) {
          compositionEndData = null;
          compositionFirstEmitted = false;
        }

        if (currentWs.readyState === WebSocket.OPEN) {
          currentWs.send(encoder.encode(data));
        }
      });

      currentTerm.onResize(({ rows, cols }) => {
        if (currentWs.readyState === WebSocket.OPEN) {
          currentWs.send(JSON.stringify({ type: "resize", rows, cols }));
        }
      });
    }

    function doMount() {
      if (!alive || !container) return;
      if (container.clientWidth === 0 || container.clientHeight === 0) {
        requestAnimationFrame(doMount);
        return;
      }

      const { theme } = getTerminalTheme(useTerminalThemeStore.getState().themeId);

      term = new XTerm({
        cursorBlink: true,
        fontSize: 14,
        fontFamily: "'JetBrains Mono', 'Fira Code', 'Cascadia Code', monospace",
        theme,
        allowProposedApi: true,
      });

      termRef.current = term;

      const fitAddon = new FitAddon();
      term.loadAddon(fitAddon);
      term.loadAddon(new WebLinksAddon());
      const unicodeAddon = new Unicode11Addon();
      term.loadAddon(unicodeAddon);
      term.unicode.activeVersion = "11";

      // Register compositionend handler in capture phase on the container BEFORE
      // term.open() so it fires before xterm.js processes the event.
      const onCompositionEnd = ((e: CompositionEvent) => {
        compositionEndData = e.data || null;
        compositionFirstEmitted = false;
        clearTimeout(compositionTimer);
        compositionTimer = setTimeout(() => {
          compositionEndData = null;
          compositionFirstEmitted = false;
        }, 100);
      }) as EventListener;
      container.addEventListener("compositionend", onCompositionEnd, true);

      term.open(container);
      fitAddon.fit();

      const currentTerm = term;
      const encoder = new TextEncoder();

      // Respond to kitty keyboard protocol queries from Claude Code.
      // When Claude Code starts, it queries/enables the kitty protocol via CSI sequences.
      // xterm.js doesn't support it natively, so we intercept and respond manually.
      // This tells Claude Code that Shift+Enter will arrive as \x1b[13;2u.
      let kittyModeFlags = 0;

      // Handle CSI ? u — kitty protocol query (Claude Code asks "do you support this?")
      currentTerm.parser.registerCsiHandler({ prefix: "?", final: "u" }, () => {
        if (ws && ws.readyState === WebSocket.OPEN) {
          ws.send(encoder.encode(`\x1b[?${kittyModeFlags}u`));
        }
        return false;
      });

      // Handle CSI > flags u — kitty protocol push (Claude Code enables the protocol)
      currentTerm.parser.registerCsiHandler({ prefix: ">", final: "u" }, (params) => {
        kittyModeFlags = (params[0] as number) ?? 1;
        return false;
      });

      // Handle CSI < u — kitty protocol pop
      currentTerm.parser.registerCsiHandler({ prefix: "<", final: "u" }, () => {
        kittyModeFlags = 0;
        return false;
      });

      // Intercept Shift+Enter at the DOM level (capture phase) to fully prevent
      // xterm.js from also sending \r. Send kitty protocol escape sequence instead.
      const onKeyDown = (event: KeyboardEvent) => {
        if (event.key === "Enter" && event.shiftKey) {
          event.preventDefault();
          event.stopPropagation();
          if (ws && ws.readyState === WebSocket.OPEN) {
            ws.send(encoder.encode("\x1b[13;2u"));
          }
        }
      };
      container.addEventListener("keydown", onKeyDown, true);

      connectWs(currentTerm, fitAddon, encoder);

      resizeObserver = new ResizeObserver(() => {
        clearTimeout(resizeTimer);
        resizeTimer = setTimeout(() => fitAddon.fit(), 50);
      });
      resizeObserver.observe(container);
    }

    doMount();

    const cleanup = () => {
      alive = false;
      intentionalClose = true;
      clearTimeout(resizeTimer);
      clearTimeout(reconnectTimer);
      resizeObserver?.disconnect();
      if (ws && ws.readyState <= WebSocket.OPEN) {
        ws.close();
      }
      term?.dispose();
      termRef.current = null;
      term = null;
      ws = null;
    };

    // Note: container event listeners are cleaned up when term.dispose()
    // removes the terminal DOM elements, and when the container is unmounted.

    cleanupRef.current = cleanup;

    return cleanup;
  }, [sessionId, wsBaseUrl]);

  const bg = getTerminalTheme(themeId).theme.background;

  return (
    <div
      ref={containerRef}
      className="absolute inset-0"
      style={{ backgroundColor: bg }}
    />
  );
}
