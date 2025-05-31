import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  distDir: 'dist',
  trailingSlash: true,
  images: {
    unoptimized: true
  },
  async rewrites() {
    return [
      {
        source: '/graphql',
        destination: 'http://localhost:8080/graphql',
      },
    ];
  },
};

export default nextConfig;
