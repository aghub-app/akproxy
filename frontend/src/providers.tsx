import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { type ReactNode, useEffect, useState } from "react";
import { ToastProvider } from "@/components/ui/toast";
import { UpdateDialog } from "@/features/update-dialog";
import { api } from "@/lib/desktop";
import { playInteractionSound } from "@/lib/ui-sounds";
import { Events } from "@wailsio/runtime";

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
    const offStatus = Events.On("server:status", (event) => {
      client.setQueryData(["status"], event.data);
    });
    const offLogin = Events.On("login:done", () => {
      void client.invalidateQueries({ queryKey: ["accounts"] });
    });
    void api.status().then((status) => client.setQueryData(["status"], status));
    return () => {
      offStatus();
      offLogin();
    };
  }, [client]);

  return (
    <QueryClientProvider client={client}>
      <ToastProvider position="bottom-center">
        {children}
        <UpdateDialog />
      </ToastProvider>
    </QueryClientProvider>
  );
}
