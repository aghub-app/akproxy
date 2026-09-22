if (import.meta.env.DEV) {
  import("react-grab");
}

import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { RouterProvider } from "react-router";
import { router } from "@/app/router";
import { Providers } from "@/providers";
import "./index.css";

const colorScheme = window.matchMedia("(prefers-color-scheme: dark)");

function applySystemTheme() {
  const dark = colorScheme.matches;
  document.documentElement.classList.toggle("dark", dark);
  document.documentElement.style.colorScheme = dark ? "dark" : "light";
}

applySystemTheme();
colorScheme.addEventListener("change", applySystemTheme);

const root = document.getElementById("root");
if (!root) {
  throw new Error("找不到根节点");
}

createRoot(root).render(
  <StrictMode>
    <Providers>
      <RouterProvider router={router} />
    </Providers>
  </StrictMode>,
);
