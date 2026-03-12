import { useEffect, useRef, useCallback } from "react";
import { Terminal as XTerm } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import { WebLinksAddon } from "@xterm/addon-web-links";
import "@xterm/xterm/css/xterm.css";
import { useWebSocket } from "../hooks/useWebSocket";

interface TerminalProps {
  sessionId: string;
  wsBaseUrl?: string;
}

export function Terminal({ sessionId, wsBaseUrl }: TerminalProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const xtermRef = useRef<XTerm | null>(null);
  const fitAddonRef = useRef<FitAddon | null>(null);

  const wsUrl =
    wsBaseUrl ?? `ws://${window.location.hostname}:8080/ws/sessions/${sessionId}`;

  const handleMessage = useCallback((data: ArrayBuffer | string) => {
    const term = xtermRef.current;
    if (!term) return;

    if (data instanceof ArrayBuffer) {
      term.write(new Uint8Array(data));
    } else {
      term.write(data);
    }
  }, []);

  const handleOpen = useCallback(() => {
    const fitAddon = fitAddonRef.current;
    if (fitAddon) {
      fitAddon.fit();
    }
  }, []);

  const { sendBinary, sendText } = useWebSocket({
    url: wsUrl,
    onMessage: handleMessage,
    onOpen: handleOpen,
  });

  useEffect(() => {
    const container = containerRef.current;
    if (!container) return;

    const term = new XTerm({
      cursorBlink: true,
      fontSize: 14,
      fontFamily: "'JetBrains Mono', 'Fira Code', 'Cascadia Code', monospace",
      theme: {
        background: "#1a1b26",
        foreground: "#c0caf5",
        cursor: "#c0caf5",
        selectionBackground: "#33467c",
        black: "#15161e",
        red: "#f7768e",
        green: "#9ece6a",
        yellow: "#e0af68",
        blue: "#7aa2f7",
        magenta: "#bb9af7",
        cyan: "#7dcfff",
        white: "#a9b1d6",
        brightBlack: "#414868",
        brightRed: "#f7768e",
        brightGreen: "#9ece6a",
        brightYellow: "#e0af68",
        brightBlue: "#7aa2f7",
        brightMagenta: "#bb9af7",
        brightCyan: "#7dcfff",
        brightWhite: "#c0caf5",
      },
      allowProposedApi: true,
    });

    const fitAddon = new FitAddon();
    const webLinksAddon = new WebLinksAddon();

    term.loadAddon(fitAddon);
    term.loadAddon(webLinksAddon);

    term.open(container);
    fitAddon.fit();

    xtermRef.current = term;
    fitAddonRef.current = fitAddon;

    term.onData((data) => {
      const encoder = new TextEncoder();
      sendBinary(encoder.encode(data));
    });

    const resizeObserver = new ResizeObserver(() => {
      fitAddon.fit();
    });
    resizeObserver.observe(container);

    term.onResize(({ rows, cols }) => {
      sendText(JSON.stringify({ type: "resize", rows, cols }));
    });

    return () => {
      resizeObserver.disconnect();
      term.dispose();
      xtermRef.current = null;
      fitAddonRef.current = null;
    };
  }, [sessionId, sendBinary, sendText]);

  return (
    <div
      ref={containerRef}
      className="h-full w-full"
      style={{ backgroundColor: "#1a1b26" }}
    />
  );
}
