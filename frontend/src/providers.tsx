import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { type ReactNode, useEffect, useState } from "react";
import { ToastProvider } from "@/components/ui/toast";
import { api } from "@/lib/desktop";
import { playInteractionSound } from "@/lib/ui-sounds";
import { EventsOff, EventsOn } from "../wailsjs/runtime/runtime";

export function Providers({ children }: { children: ReactNode }) {
  const [client] = useState(
    () =>
      new QueryClient({
        defaultOptions: {
          queries: { retry: false, refetchOnWindowFocus: false },
        },
      }),
  );

  useEffect(() => {
    function onClick(event: MouseEvent) {
      if (event.target instanceof Element) {
        playInteractionSound(event.target);
      }
    }
    document.addEventListener("click", onClick);
    return () => document.removeEventListener("click", onClick);
  }, []);

  useEffect(() => {
    if (!window.runtime) {
      return;
    }
    EventsOn("server:status", (status) => {
      client.setQueryData(["status"], status);
    });
    EventsOn("login:done", () => {
      void client.invalidateQueries({ queryKey: ["accounts"] });
    });
    void api.status().then((status) => client.setQueryData(["status"], status));
    return () => {
      EventsOff("server:status");
      EventsOff("login:done");
    };
  }, [client]);

  return (
    <QueryClientProvider client={client}>
      <ToastProvider position="bottom-center">{children}</ToastProvider>
    </QueryClientProvider>
  );
}
