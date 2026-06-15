import { RouterProvider } from "react-router";
import { QueryClientProvider } from "@tanstack/react-query";
import { router } from "@/router";
import { queryClient } from "@/config/query-client";
import { ToastContainer } from "@/components/ui/Toast";
import { useNotificationWebSocket } from "@/features/sessions/hooks/useNotificationWebSocket";

function NotificationManager() {
  useNotificationWebSocket();
  return null;
}

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
      <NotificationManager />
      <ToastContainer />
    </QueryClientProvider>
  );
}

export default App;
