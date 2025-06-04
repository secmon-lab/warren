import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import path from "path";

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
  server: {
    proxy: {
      "/graphql": "http://localhost:8080",
      "/api": "http://localhost:8080",
    },
  },
  build: {
    outDir: "dist",
    assetsDir: "assets",
    // Ensure proper handling of SPA routing
    rollupOptions: {
      output: {
        manualChunks: {
          vendor: ["react", "react-dom"],
          apollo: ["@apollo/client", "graphql"],
          ui: [
            "@radix-ui/react-avatar",
            "@radix-ui/react-dialog",
            "@radix-ui/react-dropdown-menu",
            "@radix-ui/react-tabs",
          ],
        },
      },
    },
  },
});
