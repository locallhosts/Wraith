/** @type {import('next').NextConfig} */
const nextConfig = {
  async rewrites() { return [{ source: "/jobs", destination: "/pipeline" }]; },
  reactStrictMode: true,
};
module.exports = nextConfig;
