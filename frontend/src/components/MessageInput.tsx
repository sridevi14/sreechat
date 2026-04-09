import { useState, useRef, useEffect } from "react";
import { Smile } from "lucide-react";

interface Props {
  onSend: (content: string) => void;
  onTyping: () => void;
}

const EMOJI_GRID = [
  "😀",
  "😂",
  "🥰",
  "😍",
  "😊",
  "😢",
  "😭",
  "🙏",
  "👍",
  "👎",
  "❤️",
  "🔥",
  "✨",
  "🎉",
  "💯",
  "🤔",
  "😅",
  "👋",
  "🙌",
  "✅",
  "❌",
  "⭐",
  "🙈",
  "😴",
];

export default function MessageInput({ onSend, onTyping }: Props) {
  const [text, setText] = useState("");
  const [pickerOpen, setPickerOpen] = useState(false);
  const typingTimeout = useRef<ReturnType<typeof setTimeout>>();
  const inputRef = useRef<HTMLInputElement>(null);
  const pickerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!pickerOpen) return;
    const onDoc = (e: MouseEvent) => {
      if (
        pickerRef.current &&
        !pickerRef.current.contains(e.target as Node) &&
        !(e.target as HTMLElement).closest("[data-emoji-toggle]")
      ) {
        setPickerOpen(false);
      }
    };
    document.addEventListener("mousedown", onDoc);
    return () => document.removeEventListener("mousedown", onDoc);
  }, [pickerOpen]);

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setText(e.target.value);
    clearTimeout(typingTimeout.current);
    onTyping();
    typingTimeout.current = setTimeout(() => {}, 2000);
  };

  const insertEmoji = (emoji: string) => {
    const el = inputRef.current;
    if (el) {
      const start = el.selectionStart ?? text.length;
      const end = el.selectionEnd ?? text.length;
      const next = text.slice(0, start) + emoji + text.slice(end);
      setText(next);
      requestAnimationFrame(() => {
        el.focus();
        const pos = start + emoji.length;
        el.setSelectionRange(pos, pos);
      });
    } else {
      setText((t) => t + emoji);
    }
    onTyping();
  };

  const handleSend = () => {
    if (!text.trim()) return;
    onSend(text.trim());
    setText("");
    setPickerOpen(false);
  };

  return (
    <div className="px-6 py-4 bg-slate-800 border-t border-slate-700">
      <div className="flex gap-2 items-stretch">
        <input
          ref={inputRef}
          type="text"
          value={text}
          onChange={handleChange}
          onKeyDown={(e) => {
            if (e.key === "Enter") handleSend();
            if (e.key === "Escape") setPickerOpen(false);
          }}
          placeholder="Type a message..."
          className="flex-1 min-w-0 px-4 py-3 bg-slate-700 rounded-xl text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-indigo-500"
        />
        <div className="relative flex shrink-0 gap-2" ref={pickerRef}>
          <button
            type="button"
            data-emoji-toggle
            onClick={() => setPickerOpen((o) => !o)}
            className="px-3 py-3 bg-slate-700 hover:bg-slate-600 text-amber-300 rounded-xl transition-colors border border-slate-600/50"
            title="Emoji"
            aria-expanded={pickerOpen}
            aria-label="Open emoji picker"
          >
            <Smile className="w-5 h-5" strokeWidth={2} />
          </button>
          {pickerOpen && (
            <div
              className="absolute bottom-full right-0 mb-2 p-2 rounded-xl bg-slate-800 border border-slate-600 shadow-xl z-50 w-[220px] grid grid-cols-6 gap-1"
              role="listbox"
            >
              {EMOJI_GRID.map((em) => (
                <button
                  key={em}
                  type="button"
                  role="option"
                  className="text-xl p-1.5 rounded-lg hover:bg-slate-700 transition-colors leading-none"
                  onClick={() => {
                    insertEmoji(em);
                    setPickerOpen(false);
                  }}
                >
                  {em}
                </button>
              ))}
            </div>
          )}
          <button
            type="button"
            onClick={handleSend}
            disabled={!text.trim()}
            className="px-5 py-3 bg-indigo-600 hover:bg-indigo-700 disabled:opacity-40 disabled:cursor-not-allowed text-white font-medium rounded-xl transition-colors"
          >
            Send
          </button>
        </div>
      </div>
    </div>
  );
}
