import { defineConfig } from "astro/config";
import sitemap from "@astrojs/sitemap";

export default defineConfig({
  site: "https://akproxy.akr.moe",
  // Static assets keep their existing URLs: favicon.svg and public/*.
  publicDir: "./static",
  compressHTML: false,
  i18n: {
    defaultLocale: "zh",
    locales: ["zh", "en"],
    routing: {
      prefixDefaultLocale: false,
    },
  },
  integrations: [
    sitemap({
      i18n: {
        defaultLocale: "zh",
        locales: { zh: "zh", en: "en" },
      },
    }),
  ],
});
