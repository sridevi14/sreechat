import { useEffect } from "react";
import { useAuthStore } from "./store/authStore";
import { useGlobalPresence } from "./hooks/useGlobalPresence";
import AuthPage from "./components/AuthPage";
import Sidebar from "./components/Sidebar";
import ChatWindow from "./components/ChatWindow";

function AuthenticatedShell() {
  useGlobalPresence();
  return (
    <div className="h-screen flex">
      <Sidebar />
      <ChatWindow />
    </div>
  );
}

export default function App() {
  const { isAuthenticated, hydrate } = useAuthStore();

  useEffect(() => {
    hydrate();
  }, [hydrate]);

  if (!isAuthenticated) {
    return <AuthPage />;
  }

  return <AuthenticatedShell />;
}
