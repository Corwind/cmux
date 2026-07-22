import { useQuery } from "@tanstack/react-query";
import { fetchHarnesses } from "../services/harnesses-api";

export const harnessKeys = {
  all: ["harnesses"] as const,
};

export function useHarnesses() {
  return useQuery({
    queryKey: harnessKeys.all,
    queryFn: fetchHarnesses,
  });
}
