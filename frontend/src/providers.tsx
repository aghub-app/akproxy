import { QueryCache, QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { type ReactNode, useEffect, useState } from "react";
import { ToastProvider, toastManager } from "@/components/ui/toast";
import { UpdateDialog } from "@/features/update-dialog";
import { errorText, type Status, type UpdatePrefsStatus } from "@/lib/desktop";
import { playInteractionSound } from "@/lib/ui-sounds";
import { Events } from "@wailsio/runtime";
import { PresentationProvider } from "@/presentation";

export function Providers({ children }: { children: ReactNode }) {
  const [client] = useState(
    () => {
      const reported = new WeakSet<object>();
      return new QueryClient({
        queryCache: new QueryCache({
          onError: (error, query) => {
            if (!reported.has(query)) {
              reported.add(query);
              toastManager.add({ title: errorText(error), type: "error" });
            }
          },
          onSuccess: (_data, query) => reported.delete(query),
        }),
        defaultOptions: {
          queries: { retry: false, refetchOnWindowFocus: false },
        },
      });
    },
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
      const next = event.data as Status;
      const previous = client.getQueryData<Status>(["status"]);
      client.setQueryData(["status"], next);
      if (next.error && previous?.running && !next.running) {
        toastManager.add({ title: errorText(next.error), type: "error" });
      }
    });
    const offLogin = Events.On("login:done", () => {
      void client.invalidateQueries({ queryKey: ["accounts"] });
    });
    const offUpdateChecked = Events.On("updates:checked", (event) => {
      client.setQueryData<UpdatePrefsStatus>(["update-prefs"], event.data as UpdatePrefsStatus);
    });
    const offUpdateError = Events.On("updates:check-error", (event) => {
      toastManager.add({ title: errorText(event.data), type: "error" });
    });
    return () => {
      offStatus();
      offLogin();
      offUpdateChecked();
      offUpdateError();
    };
  }, [client]);

  return (
    <QueryClientProvider client={client}>
      <PresentationProvider>
        <ToastProvider position="bottom-center">
          {children}
          <UpdateDialog />
        </ToastProvider>
      </PresentationProvider>
    </QueryClientProvider>
  );
}
