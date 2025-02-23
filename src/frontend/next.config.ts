import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  /* config options here */
  env: {
    backendUrl: process.env.BACKEND_URL
  },
  eslint: {
    ignoreDuringBuilds: true
  },
  output: 'standalone',
};

export default nextConfig;
