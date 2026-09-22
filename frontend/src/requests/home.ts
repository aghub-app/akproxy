import { queryOptions } from "@tanstack/react-query";
import { api } from "@/lib/desktop";
import { queryKeys } from "./keys";

export const homeQueryOptions = () => queryOptions({
  queryKey: queryKeys.home.snapshot(),
  queryFn: api.home,
  refetchInterval: 2000,
  staleTime: 0,
});

export const homeModelsQueryOptions = (address: string | null) => queryOptions({
  queryKey: queryKeys.home.models(address),
  queryFn: api.homeModels,
  enabled: address !== null,
  refetchInterval: 5000,
  staleTime: 0,
});
