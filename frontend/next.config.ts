import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  distDir: "dist",
  // 開発時はrewritesを有効にし、本番ビルド時は静的エクスポートを使用
  ...(process.env.NODE_ENV === "production" &&
  process.env.NEXT_EXPORT === "true"
    ? {
        output: "export",
        // Ensure query parameters work with static export
        exportPathMap: async function (defaultPathMap) {
          return {
            ...defaultPathMap,
            "/tickets": { page: "/tickets" },
          };
        },
      }
    : {}),
  trailingSlash: true,
  images: {
    unoptimized: true,
  },
  // 開発時のみrewritesを有効にする
  ...(process.env.NODE_ENV === "development"
    ? {
        async rewrites() {
          return [
            {
              source: "/graphql",
              destination: "http://localhost:8080/graphql",
            },
            {
              source: "/api/auth/:path*",
              destination: "http://localhost:8080/api/auth/:path*",
            },
            {
              source: "/api/user/:path*",
              destination: "http://localhost:8080/api/user/:path*",
            },
          ];
        },
      }
    : {}),
};

export default nextConfig;
