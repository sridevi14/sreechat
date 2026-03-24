import { useEffect } from "react";
import { useAuthStore } from "./store/authStore";
import AuthPage from "./components/AuthPage";
import Sidebar from "./components/Sidebar";
import ChatWindow from "./components/ChatWindow";

export default function App() {
  const { isAuthenticated, hydrate } = useAuthStore();

  useEffect(() => {
    hydrate();
  }, [hydrate]);

  if (!isAuthenticated) {
    return <AuthPage />;
  }

  return (
    <div className="h-screen flex">
      <Sidebar />
      <ChatWindow />
    </div>
  );
}
