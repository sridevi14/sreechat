import { create } from "zustand";
import { Room, Message, roomAPI } from "../api/client";

interface ChatState {
  rooms: Room[];
  activeRoom: Room | null;
  messages: Record<string, Message[]>;
  typingUsers: Record<string, string[]>;

  setRooms: (rooms: Room[]) => void;
  setActiveRoom: (room: Room | null) => void;
  fetchRooms: () => Promise<void>;
  fetchMessages: (roomId: string, afterSeq?: number) => Promise<void>;
  addMessage: (roomId: string, message: Message) => void;
  setTyping: (roomId: string, username: string, isTyping: boolean) => void;
}

const typingTimers: Record<string, ReturnType<typeof setTimeout>> = {};

export const useChatStore = create<ChatState>((set, get) => ({
  rooms: [],
  activeRoom: null,
  messages: {},
  typingUsers: {},

  setRooms: (rooms) => set({ rooms }),
  setActiveRoom: (room) => set({ activeRoom: room }),

  fetchRooms: async () => {
    try {
      const { data } = await roomAPI.list();
      set({ rooms: data || [] });
    } catch {
      // ignore if not authenticated yet
    }
  },

  fetchMessages: async (roomId, afterSeq) => {
    const { data } = await roomAPI.messages(roomId, afterSeq, 50);
    const existing = get().messages[roomId] || [];
    const reversed = (data || []).reverse();

    const seqSet = new Set(existing.map((m) => m.seq));
    const newMsgs = reversed.filter((m) => !seqSet.has(m.seq));
    const merged = [...newMsgs, ...existing].sort((a, b) => a.seq - b.seq);

    set({ messages: { ...get().messages, [roomId]: merged } });
  },

  addMessage: (roomId, message) => {
    const existing = get().messages[roomId] || [];
    if (existing.some((m) => m.seq === message.seq)) return;
    const updated = [...existing, message].sort((a, b) => a.seq - b.seq);

    if (existing.length > 0) {
      const lastSeq = existing[existing.length - 1].seq;
      if (message.seq > lastSeq + 1) {
        roomAPI
          .messages(roomId, message.seq, message.seq - lastSeq)
          .then(({ data }) => {
            const all = get().messages[roomId] || [];
            const seqSet = new Set(all.map((m) => m.seq));
            const fill = (data || []).filter((m) => !seqSet.has(m.seq));
            set({
              messages: {
                ...get().messages,
                [roomId]: [...all, ...fill].sort((a, b) => a.seq - b.seq),
              },
            });
          });
      }
    }

    set({ messages: { ...get().messages, [roomId]: updated } });
  },

  setTyping: (roomId, username, isTypingNow) => {
    const timerKey = `${roomId}:${username}`;
    const current = get().typingUsers[roomId] || [];

    if (isTypingNow) {
      if (!current.includes(username)) {
        set({
          typingUsers: {
            ...get().typingUsers,
            [roomId]: [...current, username],
          },
        });
      }
      // Auto-clear after 3s if no stop event arrives
      clearTimeout(typingTimers[timerKey]);
      typingTimers[timerKey] = setTimeout(() => {
        const cur = get().typingUsers[roomId] || [];
        set({
          typingUsers: {
            ...get().typingUsers,
            [roomId]: cur.filter((u) => u !== username),
          },
        });
      }, 3000);
    } else {
      clearTimeout(typingTimers[timerKey]);
      set({
        typingUsers: {
          ...get().typingUsers,
          [roomId]: current.filter((u) => u !== username),
        },
      });
    }
  },
}));
