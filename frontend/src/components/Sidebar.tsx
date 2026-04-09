import { useEffect, useState } from "react";
import { Circle, Clock } from "lucide-react";
import { useChatStore } from "../store/chatStore";
import { useAuthStore } from "../store/authStore";
import { Room, User, userAPI, roomAPI } from "../api/client";
import { getDirectPeerId, formatLastSeen } from "../utils/presence";

export default function Sidebar() {
  const {
    rooms,
    activeRoom,
    setActiveRoom,
    fetchRooms,
    typingUsers,
    presenceByUserId,
    fetchPresenceForUsers,
  } = useChatStore();
  const { user, logout } = useAuthStore();
  const [showSearch, setShowSearch] = useState(false);
  const [phoneQuery, setPhoneQuery] = useState("");
  const [searchResults, setSearchResults] = useState<User[]>([]);
  const [searching, setSearching] = useState(false);

  useEffect(() => {
    fetchRooms();
  }, [fetchRooms]);

  useEffect(() => {
    if (!user) return;
    const peerIds = rooms
      .map((r) => getDirectPeerId(r, user.id))
      .filter((id): id is string => !!id);
    const unique = [...new Set(peerIds)];
    if (unique.length === 0) return;

    fetchPresenceForUsers(unique);
    const t = setInterval(() => fetchPresenceForUsers(unique), 30_000);
    return () => clearInterval(t);
  }, [rooms, user, fetchPresenceForUsers]);

  const handleSearch = async () => {
    if (!phoneQuery.trim()) return;
    setSearching(true);
    try {
      const { data } = await userAPI.search(phoneQuery.trim());
      setSearchResults(data || []);
    } catch {
      setSearchResults([]);
    } finally {
      setSearching(false);
    }
  };

  const handleStartChat = async (peer: User) => {
    try {
      const { data } = await roomAPI.startDirect(peer.id);
      setShowSearch(false);
      setPhoneQuery("");
      setSearchResults([]);
      await fetchRooms();
      setActiveRoom(data);
    } catch (err) {
      console.error("start direct chat failed:", err);
    }
  };

  const getRoomTyping = (room: Room): string[] => {
    return (typingUsers[room.id] || []).filter((u) => u !== user?.username);
  };

  return (
    <div className="w-72 bg-slate-800 border-r border-slate-700 flex flex-col h-full">
      <div className="p-4 border-b border-slate-700">
        <h2 className="text-lg font-semibold text-white">SreeChat</h2>
        <p className="text-xs text-slate-400 mt-1">
          {user?.username} &middot; {user?.phone}
        </p>
      </div>

      <div className="p-3">
        <button
          onClick={() => {
            setShowSearch(!showSearch);
            setSearchResults([]);
            setPhoneQuery("");
          }}
          className="w-full py-2 text-sm bg-indigo-600 hover:bg-indigo-700 text-white rounded-lg transition-colors"
        >
          {showSearch ? "Cancel" : "+ New Chat"}
        </button>

        {showSearch && (
          <div className="mt-3 space-y-2">
            <div className="flex gap-2">
              <input
                type="tel"
                placeholder="Search by phone number"
                value={phoneQuery}
                onChange={(e) => setPhoneQuery(e.target.value)}
                onKeyDown={(e) => e.key === "Enter" && handleSearch()}
                className="flex-1 px-3 py-2 bg-slate-700 rounded-lg text-white text-sm placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-indigo-500"
              />
              <button
                onClick={handleSearch}
                disabled={searching}
                className="px-3 py-2 bg-green-600 hover:bg-green-700 disabled:opacity-50 text-white text-sm rounded-lg"
              >
                {searching ? "..." : "Go"}
              </button>
            </div>

            {searchResults.length > 0 && (
              <div className="bg-slate-700/50 rounded-lg overflow-hidden">
                {searchResults.map((result) => (
                  <button
                    key={result.id}
                    onClick={() => handleStartChat(result)}
                    className="w-full text-left px-3 py-2.5 hover:bg-slate-600/50 transition-colors border-b border-slate-600/30 last:border-0"
                  >
                    <p className="text-sm font-medium text-white">
                      {result.username}
                    </p>
                    <p className="text-xs text-slate-400">{result.phone}</p>
                  </button>
                ))}
              </div>
            )}

            {searchResults.length === 0 && phoneQuery && !searching && (
              <p className="text-xs text-slate-500 text-center py-2">
                No users found
              </p>
            )}
          </div>
        )}
      </div>

      <div className="flex-1 overflow-y-auto">
        {rooms.map((room: Room) => {
          const typing = getRoomTyping(room);
          const peerId = user ? getDirectPeerId(room, user.id) : null;
          const p = peerId ? presenceByUserId[peerId] : undefined;

          return (
            <button
              key={room.id}
              onClick={() => setActiveRoom(room)}
              className={`w-full text-left px-4 py-3 border-b border-slate-700/50 transition-colors ${
                activeRoom?.id === room.id
                  ? "bg-indigo-600/20 border-l-2 border-l-indigo-500"
                  : "hover:bg-slate-700/50"
              }`}
            >
              <div className="flex items-start justify-between gap-2">
                <p className="text-sm font-medium text-white truncate flex-1 min-w-0">
                  {room.name}
                </p>
                {room.type === "direct" && peerId && (
                  <span
                    className="shrink-0 mt-0.5"
                    title={
                      p?.online
                        ? "Online"
                        : formatLastSeen(p?.last_seen_at)
                    }
                  >
                    {p?.online ? (
                      <Circle className="w-2.5 h-2.5 fill-emerald-400 text-emerald-400" />
                    ) : (
                      <Clock className="w-3.5 h-3.5 text-slate-500" />
                    )}
                  </span>
                )}
              </div>
              {typing.length > 0 ? (
                <p className="text-xs text-green-400 italic mt-0.5">
                  {typing.join(", ")} typing...
                </p>
              ) : (
                <p className="text-xs text-slate-400 mt-0.5">
                  {room.type === "direct"
                    ? p?.online
                      ? "Online"
                      : formatLastSeen(p?.last_seen_at)
                    : `${room.members.length} members`}
                </p>
              )}
            </button>
          );
        })}
        {rooms.length === 0 && (
          <p className="text-sm text-slate-500 text-center mt-8 px-4">
            No chats yet. Search by phone number to start a conversation.
          </p>
        )}
      </div>

      <div className="p-3 border-t border-slate-700">
        <button
          onClick={logout}
          className="w-full py-2 text-sm text-slate-400 hover:text-white hover:bg-slate-700 rounded-lg transition-colors"
        >
          Sign Out
        </button>
      </div>
    </div>
  );
}
