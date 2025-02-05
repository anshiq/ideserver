import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  /* config options here */
  env: {
    backendUrl: process.env.backend_url
}
};

export default nextConfig;
