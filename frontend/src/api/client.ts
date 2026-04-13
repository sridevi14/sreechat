import axios from "axios";

const baseURL = import.meta.env.VITE_API_URL ? `${import.meta.env.VITE_API_URL}/api` : "/api";
const api = axios.create({ baseURL });

api.interceptors.request.use((config) => {
  const token = localStorage.getItem("token");
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

export interface User {
  id: string;
  username: string;
  phone: string;
  avatar: string;
}

export interface Room {
  id: string;
  name: string;
  type: string;
  members: string[];
  created_at: string;
}

export interface Message {
  id: string;
  room_id: string;
  sender_id: string;
  content: string;
  seq: number;
  created_at: string;
  username?: string;
}

export interface AuthResponse {
  token: string;
  user: User;
}

export const authAPI = {
  register: (username: string,phone: string, password: string) =>
    api.post<AuthResponse>("/auth/register", { username, phone, password }),
  login: (phone: string, password: string) =>
    api.post<AuthResponse>("/auth/login", { phone, password }),
};

export interface UserPresence {
  online: boolean;
  last_seen_at?: string;
}

export const userAPI = {
  search: (phone: string) => api.get<User[]>("/users/search", { params: { phone } }),
  /** Comma-separated user ids (max 50). Returns map id → presence. */
  presence: (userIds: string[]) =>
    api.get<Record<string, UserPresence>>("/users/presence", {
      params: { ids: userIds.join(",") },
    }),
};

/** App-wide presence: online while logged in; last seen when tab closes / logout. */
export const presenceAPI = {
  heartbeat: () => api.post("/presence/heartbeat"),
  offline: () => api.post("/presence/offline"),
};

export const roomAPI = {
  list: () => api.get<Room[]>("/rooms"),
  create: (name: string, type: string, members: string[]) =>
    api.post<Room>("/rooms", { name, type, members }),
  startDirect: (peerId: string) =>
    api.post<Room>("/rooms/direct", { peer_id: peerId }),
  messages: (roomId: string, afterSeq?: number, limit?: number) =>
    api.get<Message[]>(`/rooms/${roomId}/messages`, {
      params: { after: afterSeq, limit },
    }),
};

export default api;
