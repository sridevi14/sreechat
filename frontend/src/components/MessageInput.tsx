import { useState, useRef } from "react";

interface Props {
  onSend: (content: string) => void;
  onTyping: () => void;
}

export default function MessageInput({ onSend, onTyping }: Props) {
  const [text, setText] = useState("");
  const typingTimeout = useRef<ReturnType<typeof setTimeout>>();

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setText(e.target.value);
    clearTimeout(typingTimeout.current);
    onTyping();
    typingTimeout.current = setTimeout(() => {}, 2000);
  };

  const handleSend = () => {
    if (!text.trim()) return;
    onSend(text.trim());
    setText("");
  };

  return (
    <div className="px-6 py-4 bg-slate-800 border-t border-slate-700">
      <div className="flex gap-3">
        <input
          type="text"
          value={text}
          onChange={handleChange}
          onKeyDown={(e) => e.key === "Enter" && handleSend()}
          placeholder="Type a message..."
          className="flex-1 px-4 py-3 bg-slate-700 rounded-xl text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-indigo-500"
        />
        <button
          onClick={handleSend}
          disabled={!text.trim()}
          className="px-6 py-3 bg-indigo-600 hover:bg-indigo-700 disabled:opacity-40 disabled:cursor-not-allowed text-white font-medium rounded-xl transition-colors"
        >
          Send
        </button>
      </div>
    </div>
  );
}
