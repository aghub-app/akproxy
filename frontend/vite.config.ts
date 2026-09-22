import path from "node:path";
import tailwindcss from "@tailwindcss/vite";
import react from "@vitejs/plugin-react";
import { defineConfig, type Plugin } from "vite";
import wails from "@wailsio/runtime/plugins/vite";

// Browser-only mock of the Go bindings so the renderer can be exercised
// outside the Wails window. Never active in production builds.
const BINDING_MOCKS: Record<number, unknown> = {
  2836613335: {
    running: false, action: "start", address: "http://127.0.0.1:8317",
    allInterfaces: false, restartRequired: false, savedAddress: "http://127.0.0.1:8317", error: "",
  },
  640618321: {
    listenMode: "local", customHost: "", port: 8317,
    clientApiKeys: ["sk-abcdef1234567890abcdef1234567890"],
    proxyUrl: "", routingStrategy: "round-robin", debug: false,
  },
  2284164635: {
    prefs: { autoUpdate: true, autoCheck: true, checkIntervalHours: 24 },
    lastCheckAt: "2026-09-23 09:30", lastCheckResult: "已是最新版本",
    latestVersion: "", platform: "macOS arm64",
  },
  2431199839: "v0.1.0",
};

function wailsBrowserMock(): Plugin {
  return {
    name: "wails-browser-mock",
    transformIndexHtml() {
      return [
        {
          tag: "script",
          injectTo: "head-prepend",
          children: `(() => {
            const mocks = ${JSON.stringify(BINDING_MOCKS)};
            const prefs = mocks[2284164635];
            const realFetch = window.fetch.bind(window);
            window.fetch = (input, init) => {
              const url = String(input);
              if (url.endsWith("/wails/runtime") && init?.method === "POST") {
                const body = JSON.parse(init.body);
                const methodID = body.args?.methodID;
                if (methodID === 2750775117) {
                  Object.assign(prefs.prefs, body.args.args[0]);
                  return Promise.resolve(new Response(JSON.stringify(prefs), {
                    headers: { "Content-Type": "application/json" },
                  }));
                }
                if (methodID !== undefined && methodID in mocks) {
                  return Promise.resolve(new Response(JSON.stringify(mocks[methodID]), {
                    headers: { "Content-Type": "application/json" },
                  }));
                }
              }
              return realFetch(input, init);
            };
          })();`,
        },
      ];
    },
  };
}

export default defineConfig({
  plugins: [react(), tailwindcss(), wails("./bindings"), wailsBrowserMock()],
  server: { host: "127.0.0.1", port: 9245, strictPort: true },
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
});
