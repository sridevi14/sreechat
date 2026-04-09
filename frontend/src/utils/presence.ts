/** Other member in a 1:1 room (excludes current user). */
export function getDirectPeerId(
  room: { type: string; members: string[] },
  currentUserId: string | undefined
): string | null {
  if (room.type !== "direct" || !currentUserId) return null;
  const other = room.members.find((m) => m !== currentUserId);
  return other ?? null;
}

/** Human-readable last seen (WhatsApp-style short phrases). */
export function formatLastSeen(iso: string | undefined): string {
  if (!iso) return "last seen recently";

  const then = new Date(iso);
  if (Number.isNaN(then.getTime())) return "last seen recently";

  const now = new Date();
  const diffMs = now.getTime() - then.getTime();
  const diffM = Math.floor(diffMs / 60000);
  const diffH = Math.floor(diffMs / 3600000);
  const diffD = Math.floor(diffMs / 86400000);

  if (diffM < 1) return "last seen just now";
  if (diffM < 60) return `last seen ${diffM} min ago`;
  if (diffH < 24) return `last seen ${diffH} hr ago`;
  if (diffD === 1) return "last seen yesterday";
  if (diffD < 7) return `last seen ${diffD} days ago`;

  return `last seen ${then.toLocaleDateString(undefined, {
    month: "short",
    day: "numeric",
    hour: "numeric",
    minute: "2-digit",
  })}`;
}
