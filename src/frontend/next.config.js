/** @type {import('next').NextConfig} */
const nextConfig = {
    env: {
        backendUrl: process.env.backend_url
    }
}

module.exports = nextConfig
