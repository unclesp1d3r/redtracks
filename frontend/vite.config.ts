import path from "node:path";
import { defineConfig, type Plugin } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";
import wails from "@wailsio/runtime/plugins/vite";

// Content Security Policy for production builds only. Vite's dev server injects an
// inline React refresh script that script-src 'self' would block, so the policy is
// added at build time rather than in index.html. 'unsafe-inline' styles are needed
// by the <style> block Mermaid emits inside its SVG and by the inline style=
// positioning Base UI popovers and selects set; ws:/wss: is the Wails event
// channel in server mode. frame-ancestors is omitted: it is ignored in a meta CSP.
const CSP = [
  "default-src 'self'",
  "script-src 'self'",
  "style-src 'self' 'unsafe-inline'",
  "img-src 'self' data: blob:",
  "font-src 'self' data:",
  "connect-src 'self' ws: wss:",
  "object-src 'none'",
  "base-uri 'self'",
  "form-action 'self'",
].join("; ");

function cspPlugin(): Plugin {
  return {
    name: "redtracks-csp",
    apply: "build",
    transformIndexHtml(html) {
      return {
        html,
        tags: [{ tag: "meta", attrs: { "http-equiv": "Content-Security-Policy", content: CSP }, injectTo: "head-prepend" }],
      };
    },
  };
}

export default defineConfig({
  resolve: {
    alias: { "@": path.resolve(import.meta.dirname, "./src") },
  },
  server: {
    host: "127.0.0.1",
    port: Number(process.env.WAILS_VITE_PORT) || 9245,
    strictPort: true,
  },
  plugins: [react(), tailwindcss(), wails("./bindings"), cspPlugin()],
});
