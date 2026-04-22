/** @type {import('next').NextConfig} */
const nextConfig = {
  outputFileTracingRoot: import.meta.dirname,
  // Production API URLs are baked in at build time via NEXT_PUBLIC_ env vars
}

export default nextConfig

