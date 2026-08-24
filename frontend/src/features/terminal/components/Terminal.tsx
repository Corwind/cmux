import { useEffect, useRef } from "react";
import { Terminal as XTerm, type IDisposable } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import { WebLinksAddon } from "@xterm/addon-web-links";
import { Unicode11Addon } from "@xterm/addon-unicode11";
import { WebglAddon } from "@xterm/addon-webgl";
import { getTerminalTheme } from "../themes";
import { useTerminalThemeStore } from "../stores/terminal-theme.store";
import { useNotificationStore } from "@/features/sessions/stores/notification.store";
import { useSessionsStore } from "@/features/sessions/stores/sessions.store";
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
  const fontFamily = useTerminalThemeStore((s) => s.fontFamily);
  const activeSessionId = useSessionsStore((s) => s.activeSessionId);

  // Apply theme/font changes to a running terminal without remounting
  useEffect(() => {
    if (termRef.current) {
      const { theme } = getTerminalTheme(themeId);
      termRef.current.options.theme = theme;
    }
  }, [themeId]);

  useEffect(() => {
    if (termRef.current) {
      termRef.current.options.fontFamily = fontFamily;
    }
  }, [fontFamily]);

  useEffect(() => {
    if (activeSessionId === sessionId) {
      useNotificationStore.getState().clearNotification(sessionId);
    }
  }, [activeSessionId, sessionId]);

  useEffect(() => {
    // Clean up previous instance immediately
    cleanupRef.current?.();

    const container = containerRef.current;
    if (!container) return;

    let term: XTerm | null = null;
    let ws: WebSocket | null = null;
    let resizeObserver: ResizeObserver | null = null;
    let dataDisposable: IDisposable | null = null;
    let terminalResizeDisposable: IDisposable | null = null;
    let onKeyDown: ((event: KeyboardEvent) => void) | null = null;
    let resizeTimer: ReturnType<typeof setTimeout>;
    let reconnectTimer: ReturnType<typeof setTimeout>;
    let alive = true;
    let intentionalClose = false;
    let lastResizeSocket: WebSocket | null = null;
    let lastResizeRows = 0;
    let lastResizeCols = 0;

    const wsProtocol = window.location.protocol === "https:" ? "wss:" : "ws:";
    const wsUrl =
      wsBaseUrl ??
      `${wsProtocol}//${window.location.host}/ws/sessions/${sessionId}`;

    function sendResize(rows: number, cols: number) {
      const currentWs = ws;
      if (!currentWs || currentWs.readyState !== WebSocket.OPEN) return;
      if (
        currentWs === lastResizeSocket &&
        rows === lastResizeRows &&
        cols === lastResizeCols
      ) {
        return;
      }
      lastResizeSocket = currentWs;
      lastResizeRows = rows;
      lastResizeCols = cols;
      currentWs.send(JSON.stringify({ type: "resize", rows, cols }));
    }

    function connectWs(currentTerm: XTerm, fitAddon: FitAddon) {
      if (!alive) return;

      ws = new WebSocket(wsUrl);
      ws.binaryType = "arraybuffer";
      const currentWs = ws;

      currentWs.onopen = () => {
        if (!alive || ws !== currentWs) return;
        fitAddon.fit();
        sendResize(currentTerm.rows, currentTerm.cols);
      };

      currentWs.onmessage = (event: MessageEvent) => {
        if (!alive || ws !== currentWs) return;
        if (event.data instanceof ArrayBuffer) {
          currentTerm.write(new Uint8Array(event.data));
        } else if (event.data instanceof Blob) {
          event.data.arrayBuffer().then((buf) => {
            if (alive && ws === currentWs)
              currentTerm.write(new Uint8Array(buf));
          });
        } else {
          currentTerm.write(event.data as string);
        }
      };

      currentWs.onclose = () => {
        if (!alive || intentionalClose || ws !== currentWs) return;
        reconnectTimer = setTimeout(
          () => connectWs(currentTerm, fitAddon),
          1000,
        );
      };
    }

    function doMount() {
      if (!alive || !container) return;
      if (container.clientWidth === 0 || container.clientHeight === 0) {
        requestAnimationFrame(doMount);
        return;
      }

      const { theme } = getTerminalTheme(
        useTerminalThemeStore.getState().themeId,
      );
      const currentFontFamily = useTerminalThemeStore.getState().fontFamily;

      term = new XTerm({
        cursorBlink: true,
        fontSize: 14,
        fontFamily: currentFontFamily,
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
      term.open(container);

      // WebGL renderer: better glyph positioning and font-fallback accuracy,
      // fixes spacing issues with Powerline/Nerd Font glyphs and symbols like ⎇.
      try {
        const webgl = new WebglAddon();
        webgl.onContextLoss(() => webgl.dispose());
        term.loadAddon(webgl);
      } catch {
        // WebGL unavailable (headless, older GPU) — fall back to canvas renderer
      }

      fitAddon.fit();

      const currentTerm = term;
      const encoder = new TextEncoder();

      // These listeners belong to the terminal, not to an individual socket.
      // Keeping them outside connectWs prevents one extra pair from accumulating
      // after every reconnect and lets them always target the latest socket.
      dataDisposable = currentTerm.onData((data) => {
        const currentWs = ws;
        if (currentWs?.readyState === WebSocket.OPEN) {
          currentWs.send(encoder.encode(data));
        }
      });
      terminalResizeDisposable = currentTerm.onResize(({ rows, cols }) => {
        sendResize(rows, cols);
      });

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
      currentTerm.parser.registerCsiHandler(
        { prefix: ">", final: "u" },
        (params) => {
          kittyModeFlags = (params[0] as number) ?? 1;
          return false;
        },
      );

      // Handle CSI < u — kitty protocol pop
      currentTerm.parser.registerCsiHandler({ prefix: "<", final: "u" }, () => {
        kittyModeFlags = 0;
        return false;
      });

      // Intercept Shift+Enter at the DOM level (capture phase) to fully prevent
      // xterm.js from also sending \r. Send kitty protocol escape sequence instead.
      onKeyDown = (event: KeyboardEvent) => {
        if (event.key === "Enter" && event.shiftKey) {
          event.preventDefault();
          event.stopPropagation();
          if (ws && ws.readyState === WebSocket.OPEN) {
            ws.send(encoder.encode("\x1b[13;2u"));
          }
        }
      };
      container.addEventListener("keydown", onKeyDown, true);

      connectWs(currentTerm, fitAddon);

      resizeObserver = new ResizeObserver(() => {
        clearTimeout(resizeTimer);
        resizeTimer = setTimeout(() => fitAddon.fit(), 50);
      });
      resizeObserver.observe(container);
    }

    doMount();

    const cleanup = () => {
      if (!alive) return;
      alive = false;
      intentionalClose = true;
      clearTimeout(resizeTimer);
      clearTimeout(reconnectTimer);
      resizeObserver?.disconnect();
      dataDisposable?.dispose();
      terminalResizeDisposable?.dispose();
      if (onKeyDown) {
        container.removeEventListener("keydown", onKeyDown, true);
      }
      if (ws && ws.readyState <= WebSocket.OPEN) {
        ws.close();
      }
      term?.dispose();
      termRef.current = null;
      term = null;
      ws = null;
    };

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
