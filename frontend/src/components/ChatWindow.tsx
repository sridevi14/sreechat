import { useEffect, useRef } from "react";
import { Circle, Clock, Users } from "lucide-react";
import { useChatStore } from "../store/chatStore";
import { useAuthStore } from "../store/authStore";
import { useWebSocket } from "../hooks/useWebSocket";
import { getDirectPeerId, formatLastSeen } from "../utils/presence";
import MessageInput from "./MessageInput";

function formatMessageTime(iso: string) {
  const dt = new Date(iso);
  if (Number.isNaN(dt.getTime())) return "";
  return dt.toLocaleTimeString([], { hour: "numeric", minute: "2-digit" });
}

export default function ChatWindow() {
  const activeRoom = useChatStore((s) => s.activeRoom);
  const messages = useChatStore((s) =>
    activeRoom ? s.messages[activeRoom.id] || [] : []
  );
  const typingUsers = useChatStore((s) =>
    activeRoom ? s.typingUsers[activeRoom.id] || [] : []
  );
  const presenceByUserId = useChatStore((s) => s.presenceByUserId);
  const fetchMessages = useChatStore((s) => s.fetchMessages);
  const fetchPresenceForUsers = useChatStore((s) => s.fetchPresenceForUsers);
  const { user, token } = useAuthStore();
  const { sendMessage, sendTyping } = useWebSocket(
    activeRoom?.id || null,
    token
  );
  const bottomRef = useRef<HTMLDivElement>(null);

  const othersTyping = typingUsers.filter((u) => u !== user?.username);
  const peerId =
    activeRoom && user ? getDirectPeerId(activeRoom, user.id) : null;
  const peerPresence = peerId ? presenceByUserId[peerId] : undefined;

  useEffect(() => {
    if (activeRoom) {
      fetchMessages(activeRoom.id);
    }
  }, [activeRoom, fetchMessages]);

  useEffect(() => {
    if (peerId) {
      fetchPresenceForUsers([peerId]);
    }
  }, [peerId, fetchPresenceForUsers]);

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages]);

  if (!activeRoom) {
    return (
      <div className="flex-1 flex items-center justify-center bg-slate-900">
        <div className="text-center">
          <div className="text-6xl mb-4 opacity-20">💬</div>
          <h2 className="text-xl font-semibold text-slate-400">
            Select a chat to start messaging
          </h2>
          <p className="text-slate-500 mt-2 text-sm">
            Search by phone number to find someone
          </p>
        </div>
      </div>
    );
  }

  const initial = activeRoom.name.trim().charAt(0).toUpperCase() || "?";

  const subtitle = () => {
    if (othersTyping.length > 0) {
      return (
        <span className="text-green-400 italic">
          {othersTyping.join(", ")} typing...
        </span>
      );
    }
    if (activeRoom.type === "group") {
      return (
        <span className="inline-flex items-center gap-1.5 text-slate-400">
          <Users className="w-3.5 h-3.5 shrink-0" aria-hidden />
          {activeRoom.members.length} members
        </span>
      );
    }
    if (peerPresence?.online) {
      return (
        <span className="inline-flex items-center gap-1.5 text-emerald-400/90">
          <Circle className="w-2.5 h-2.5 fill-current shrink-0" aria-hidden />
          online
        </span>
      );
    }
    return (
      <span className="inline-flex items-center gap-1.5 text-slate-400">
        <Clock className="w-3.5 h-3.5 shrink-0 opacity-80" aria-hidden />
        {formatLastSeen(peerPresence?.last_seen_at)}
      </span>
    );
  };

  const statusBadge =
    activeRoom.type === "direct" && peerPresence?.online ? (
      <span
        className="flex h-9 w-9 items-center justify-center rounded-full bg-slate-700 text-sm font-semibold text-white ring-2 ring-emerald-500/80"
        title="Online"
      >
        {initial}
      </span>
    ) : (
      <span className="flex h-9 w-9 items-center justify-center rounded-full bg-slate-700 text-sm font-semibold text-white ring-2 ring-slate-600">
        {initial}
      </span>
    );

  return (
    <div className="flex-1 flex flex-col bg-slate-900">
      <div className="px-6 py-4 bg-slate-800 border-b border-slate-700 flex items-center gap-4">
        {statusBadge}
        <div className="min-w-0 flex-1">
          <h2 className="text-lg font-semibold text-white truncate">
            {activeRoom.name}
          </h2>
          <p className="text-xs mt-0.5">{subtitle()}</p>
        </div>
      </div>

      <div className="flex-1 overflow-y-auto px-6 py-4 space-y-3">
        {messages.map((msg) => {
          const isOwn = msg.sender_id === user?.id;
          return (
            <div
              key={`${msg.room_id}-${msg.seq}`}
              className={`flex ${isOwn ? "justify-end" : "justify-start"}`}
            >
              <div
                className={`max-w-[70%] px-4 py-2.5 rounded-2xl ${
                  isOwn
                    ? "bg-indigo-600 text-white rounded-br-md"
                    : "bg-slate-800 text-slate-200 rounded-bl-md"
                }`}
              >
                {!isOwn && (
                  <p className="text-xs text-indigo-400 font-medium mb-1">
                    {msg.username || msg.sender_id.slice(-6)}
                  </p>
                )}
                <p className="text-sm leading-relaxed">{msg.content}</p>
                <p
                  className={`text-[10px] mt-1 text-right ${
                    isOwn ? "text-indigo-200/90" : "text-slate-400"
                  }`}
                >
                  {formatMessageTime(msg.created_at)}
                </p>
              </div>
            </div>
          );
        })}
        <div ref={bottomRef} />
      </div>

      <MessageInput onSend={sendMessage} onTyping={sendTyping} />
    </div>
  );
}
