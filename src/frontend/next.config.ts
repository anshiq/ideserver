import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  /* config options here */
  env: {
    backendUrl: process.env.BACKEND_URL
  },
  eslint: {
    ignoreDuringBuilds: true
  }
};

export default nextConfig;
