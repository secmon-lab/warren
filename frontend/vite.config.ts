import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import path from "path";

// Plugin to preserve .gitkeep file during build
function preserveGitkeep() {
  return {
    name: "preserve-gitkeep",
    generateBundle() {
      // This will be called after the bundle is generated but before it's written
      this.emitFile({
        type: "asset",
        fileName: ".gitkeep",
        source: "",
      });
    },
  };
}

// https://vite.dev/config/
export default defineConfig({
  plugins: [react(), preserveGitkeep()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
  server: {
    host: "127.0.0.1",
    // Explicitly allow Tailscale hosts and other development hosts
    // Note: Vite 6.4.1 may have issues with allowedHosts configuration
    allowedHosts: [
      ".ts.net",
    ],
    proxy: {
      "/graphql": "http://localhost:8080",
      "/api": "http://localhost:8080",
      "/ws": {
        target: "http://localhost:8080",
        ws: true,
        changeOrigin: true,
        rewrite: (path) => path,
      },
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
