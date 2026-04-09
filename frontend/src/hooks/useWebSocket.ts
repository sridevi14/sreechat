import { useEffect, useRef, useCallback } from "react";
import { useChatStore } from "../store/chatStore";
import { Message } from "../api/client";

interface WSMessage {
  type: string;
  room_id: string;
  payload: {
    sender_id?: string;
    username?: string;
    content?: string;
    seq?: number;
    created_at?: string;
    is_typing?: boolean;
    user_id?: string;
    online?: boolean;
    last_seen_at?: string;
  };
}

export function useWebSocket(roomId: string | null, token: string | null) {
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimer = useRef<ReturnType<typeof setTimeout>>();
  const typingTimeout = useRef<ReturnType<typeof setTimeout>>();
  const isTyping = useRef(false);
  const addMessage = useChatStore((s) => s.addMessage);
  const setTyping = useChatStore((s) => s.setTyping);
  const setPresence = useChatStore((s) => s.setPresence);
  const fetchRooms = useChatStore((s) => s.fetchRooms);

  const connect = useCallback(() => {
    if (!roomId || !token) return;

    const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
    const url = `${protocol}//${window.location.host}/ws?token=${token}&room_id=${roomId}`;
    const ws = new WebSocket(url);
    wsRef.current = ws;

    ws.onmessage = (event) => {
      const data: WSMessage = JSON.parse(event.data);
      switch (data.type) {
        case "message": {
          const msg: Message = {
            id: "",
            room_id: data.room_id,
            sender_id: data.payload.sender_id!,
            content: data.payload.content!,
            seq: data.payload.seq!,
            created_at: data.payload.created_at || new Date().toISOString(),
            username: data.payload.username || "",
          };
          addMessage(data.room_id, msg);
          fetchRooms();
          break;
        }
        case "typing":
          setTyping(
            data.room_id,
            data.payload.username!,
            data.payload.is_typing!
          );
          break;
        case "presence": {
          const uid = data.payload.user_id;
          if (!uid) break;
          setPresence(uid, {
            online: !!data.payload.online,
            last_seen_at: data.payload.last_seen_at,
          });
          break;
        }
      }
    };

    ws.onclose = () => {
      reconnectTimer.current = setTimeout(connect, 2000);
    };

    ws.onerror = () => {
      ws.close();
    };
  }, [roomId, token, addMessage, setTyping, setPresence, fetchRooms]);

  useEffect(() => {
    connect();
    return () => {
      clearTimeout(reconnectTimer.current);
      clearTimeout(typingTimeout.current);
      if (isTyping.current && wsRef.current?.readyState === WebSocket.OPEN) {
        wsRef.current.send(
          JSON.stringify({
            type: "typing",
            room_id: roomId || "",
            payload: { is_typing: false },
          })
        );
      }
      wsRef.current?.close();
    };
  }, [connect, roomId]);

  const sendMessage = useCallback(
    (content: string) => {
      if (!wsRef.current || wsRef.current.readyState !== WebSocket.OPEN) return;
      const msg: WSMessage = {
        type: "message",
        room_id: roomId || "",
        payload: { content },
      };
      wsRef.current.send(JSON.stringify(msg));
      isTyping.current = false;
      clearTimeout(typingTimeout.current);
    },
    [roomId]
  );

  const sendTyping = useCallback(() => {
    if (!wsRef.current || wsRef.current.readyState !== WebSocket.OPEN) return;

    if (!isTyping.current) {
      isTyping.current = true;
      wsRef.current.send(
        JSON.stringify({
          type: "typing",
          room_id: roomId || "",
          payload: { is_typing: true },
        })
      );
    }

    clearTimeout(typingTimeout.current);
    typingTimeout.current = setTimeout(() => {
      isTyping.current = false;
      if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
        wsRef.current.send(
          JSON.stringify({
            type: "typing",
            room_id: roomId || "",
            payload: { is_typing: false },
          })
        );
      }
    }, 2000);
  }, [roomId]);

  return { sendMessage, sendTyping };
}
