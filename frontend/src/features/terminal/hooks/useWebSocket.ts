import { useCallback, useEffect, useRef } from "react";

interface UseWebSocketOptions {
  url: string;
  onMessage: (data: ArrayBuffer | string) => void;
  onOpen?: () => void;
  onClose?: () => void;
  onError?: (event: Event) => void;
}

export function useWebSocket({
  url,
  onMessage,
  onOpen,
  onClose,
  onError,
}: UseWebSocketOptions) {
  const wsRef = useRef<WebSocket | null>(null);
  const onMessageRef = useRef(onMessage);
  const onOpenRef = useRef(onOpen);
  const onCloseRef = useRef(onClose);
  const onErrorRef = useRef(onError);

  onMessageRef.current = onMessage;
  onOpenRef.current = onOpen;
  onCloseRef.current = onClose;
  onErrorRef.current = onError;

  useEffect(() => {
    const ws = new WebSocket(url);
    ws.binaryType = "arraybuffer";
    wsRef.current = ws;

    ws.onopen = () => onOpenRef.current?.();
    ws.onclose = () => onCloseRef.current?.();
    ws.onerror = (e) => onErrorRef.current?.(e);
    ws.onmessage = (event: MessageEvent<ArrayBuffer | string>) => {
      onMessageRef.current(event.data);
    };

    return () => {
      ws.close();
      wsRef.current = null;
    };
  }, [url]);

  const send = useCallback((data: ArrayBuffer | string) => {
    const ws = wsRef.current;
    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.send(data);
    }
  }, []);

  const sendText = useCallback(
    (data: string) => {
      send(data);
    },
    [send],
  );

  const sendBinary = useCallback(
    (data: Uint8Array) => {
      send(data.buffer as ArrayBuffer);
    },
    [send],
  );

  return { send, sendText, sendBinary, wsRef };
}
