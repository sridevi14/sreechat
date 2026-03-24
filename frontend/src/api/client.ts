import axios from "axios";

const api = axios.create({ baseURL: "/api" });

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
  email: string;
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
  register: (username: string, email: string, phone: string, password: string) =>
    api.post<AuthResponse>("/auth/register", { username, email, phone, password }),
  login: (email: string, password: string) =>
    api.post<AuthResponse>("/auth/login", { email, password }),
};

export const userAPI = {
  search: (phone: string) => api.get<User[]>("/users/search", { params: { phone } }),
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
