import type { RouteObject } from "react-router";
import { HomePage } from "@/pages/HomePage";
import { NotFoundPage } from "@/pages/NotFoundPage";

export const routes: RouteObject[] = [
  { path: "/", element: <HomePage /> },
  { path: "*", element: <NotFoundPage /> },
];
