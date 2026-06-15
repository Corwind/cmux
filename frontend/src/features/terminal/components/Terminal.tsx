import { useEffect, useRef } from "react";
import { Terminal as XTerm } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import { WebLinksAddon } from "@xterm/addon-web-links";
import { Unicode11Addon } from "@xterm/addon-unicode11";
import { WebglAddon } from "@xterm/addon-webgl";
import { useQueryClient } from "@tanstack/react-query";
import { getTerminalTheme } from "../themes";
import { useTerminalThemeStore } from "../stores/terminal-theme.store";
import { useNotificationStore } from "@/features/sessions/stores/notification.store";
import { useSessionsStore } from "@/features/sessions/stores/sessions.store";
import { sessionKeys } from "@/features/sessions";
import type { Session, NotificationEventType } from "@/features/sessions";
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
  const queryClient = useQueryClient();
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
    let oscHandler9: { dispose(): void } | null = null;
    let oscHandler777: { dispose(): void } | null = null;
    let resizeObserver: ResizeObserver | null = null;
    let resizeTimer: ReturnType<typeof setTimeout>;
    let reconnectTimer: ReturnType<typeof setTimeout>;
    let alive = true;
    let intentionalClose = false;

    const wsProtocol = window.location.protocol === "https:" ? "wss:" : "ws:";
    const wsUrl =
      wsBaseUrl ?? `${wsProtocol}//${window.location.host}/ws/sessions/${sessionId}`;

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

      function classifyOscMessage(msg: string): NotificationEventType {
        const lower = msg.toLowerCase();
        if (
          lower.includes("waiting") || lower.includes("input") ||
          lower.includes("attention") || lower.includes("permission") ||
          lower.includes("approve")
        ) {
          return "waiting_input";
        }
        if (
          lower.includes("done") || lower.includes("complete") ||
          lower.includes("finished")
        ) {
          return "task_complete";
        }
        return "generic";
      }

      // OSC 9 — ConEmu / Ghostty / iTerm2 / Kitty notification
      oscHandler9 = currentTerm.parser.registerOscHandler(9, (data) => {
        const sessions = queryClient.getQueryData<Session[]>(sessionKeys.all) ?? [];
        const sessionName = sessions.find((s) => s.id === sessionId)?.name ?? sessionId;
        useNotificationStore.getState().notify(sessionId, sessionName, data, classifyOscMessage(data));
        return false;
      });

      // OSC 777 — notify-osd / some Linux terminals
      oscHandler777 = currentTerm.parser.registerOscHandler(777, (data) => {
        const parts = data.split(";");
        const message = parts.length >= 3 ? (parts[2] ?? data) : (parts[parts.length - 1] ?? data);
        const sessions = queryClient.getQueryData<Session[]>(sessionKeys.all) ?? [];
        const sessionName = sessions.find((s) => s.id === sessionId)?.name ?? sessionId;
        useNotificationStore.getState().notify(sessionId, sessionName, message, classifyOscMessage(message));
        return false;
      });

      // BEL (\x07) — fired when Claude Code has preferredNotifChannel: "terminal_bell"
      currentTerm.onBell(() => {
        const sessions = queryClient.getQueryData<Session[]>(sessionKeys.all) ?? [];
        const sessionName = sessions.find((s) => s.id === sessionId)?.name ?? sessionId;
        useNotificationStore.getState().notify(sessionId, sessionName, "Claude needs your attention", "waiting_input");
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
      oscHandler9?.dispose();
      oscHandler777?.dispose();
      term?.dispose();
      termRef.current = null;
      term = null;
      ws = null;
    };

    // Note: container event listeners are cleaned up when term.dispose()
    // removes the terminal DOM elements, and when the container is unmounted.

    cleanupRef.current = cleanup;

    return cleanup;
  }, [sessionId, wsBaseUrl, queryClient]);

  const bg = getTerminalTheme(themeId).theme.background;

  return (
    <div
      ref={containerRef}
      className="absolute inset-0"
      style={{ backgroundColor: bg }}
    />
  );
}
