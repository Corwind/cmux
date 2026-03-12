import { RouterProvider } from "react-router";
import { QueryClientProvider } from "@tanstack/react-query";
import { router } from "@/router";
import { queryClient } from "@/config/query-client";

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  );
}

export default App;
