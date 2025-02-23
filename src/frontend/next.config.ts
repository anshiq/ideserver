import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  /* config options here */
  env: {
    backendUrl: process.env.BACKEND_URL
}
};

export default nextConfig;
