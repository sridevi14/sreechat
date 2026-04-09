import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
  server: {
    port: parseInt(process.env.VITE_PORT || "5173", 10),
    proxy: {
      "/api": process.env.VITE_API_URL || "http://localhost:8080",
      "/ws": {
        target: process.env.VITE_WS_URL || "http://localhost:8080",
        ws: true
      },
    },
  },
});
