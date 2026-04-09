import { useEffect, useRef } from "react";
import { presenceAPI } from "../api/client";

const HEARTBEAT_MS = 25_000;

/**
 * Marks the user online in Redis while this session is active (any screen, no chat required).
 * Calls offline on tab close / navigation away via keepalive fetch.
 */
export function useGlobalPresence() {
  const intervalRef = useRef<ReturnType<typeof setInterval>>();

  useEffect(() => {
    const ping = () => {
      presenceAPI.heartbeat().catch(() => {});
    };
    ping();
    intervalRef.current = setInterval(ping, HEARTBEAT_MS);

    const onPageHide = () => {
      const token = localStorage.getItem("token");
      if (!token) return;
      const url = `${window.location.origin}/api/presence/offline`;
      fetch(url, {
        method: "POST",
        headers: { Authorization: `Bearer ${token}` },
        keepalive: true,
      }).catch(() => {});
    };

    window.addEventListener("pagehide", onPageHide);

    return () => {
      window.removeEventListener("pagehide", onPageHide);
      if (intervalRef.current) clearInterval(intervalRef.current);
    };
  }, []);
}
