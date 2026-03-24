import { useEffect, useRef } from "react";
import { useChatStore } from "../store/chatStore";
import { useAuthStore } from "../store/authStore";
import { useWebSocket } from "../hooks/useWebSocket";
import MessageInput from "./MessageInput";

export default function ChatWindow() {
  const activeRoom = useChatStore((s) => s.activeRoom);
  const messages = useChatStore((s) =>
    activeRoom ? s.messages[activeRoom.id] || [] : []
  );
  const typingUsers = useChatStore((s) =>
    activeRoom ? (s.typingUsers[activeRoom.id] || []) : []
  );
  const fetchMessages = useChatStore((s) => s.fetchMessages);
  const { user, token } = useAuthStore();
  const { sendMessage, sendTyping } = useWebSocket(
    activeRoom?.id || null,
    token
  );
  const bottomRef = useRef<HTMLDivElement>(null);

  const othersTyping = typingUsers.filter((u) => u !== user?.username);

  useEffect(() => {
    if (activeRoom) {
      fetchMessages(activeRoom.id);
    }
  }, [activeRoom, fetchMessages]);

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

  return (
    <div className="flex-1 flex flex-col bg-slate-900">
      <div className="px-6 py-4 bg-slate-800 border-b border-slate-700 flex items-center justify-between">
        <div>
          <h2 className="text-lg font-semibold text-white">
            {activeRoom.name}
          </h2>
          <p className="text-xs text-slate-400">
            {othersTyping.length > 0 ? (
              <span className="text-green-400 italic">
                {othersTyping.join(", ")} typing...
              </span>
            ) : activeRoom.type === "direct" ? (
              "Direct message"
            ) : (
              `${activeRoom.members.length} members`
            )}
          </p>
        </div>
        <div className="flex items-center gap-2">
          <span className="w-2 h-2 bg-green-400 rounded-full"></span>
          <span className="text-xs text-slate-400">Online</span>
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
                {/* <p
                  className={`text-[10px] mt-1 ${
                    isOwn ? "text-indigo-200" : "text-slate-500"
                  }`}
                >
                  seq:{msg.seq}
                </p> */}
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
