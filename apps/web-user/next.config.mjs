/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  poweredByHeader: false,
  // The disposable Identity BFF fixture shares cookies on numeric loopback.
  allowedDevOrigins: ["127.0.0.1"],
  experimental: {
    cpus: 1
  },
  images: {
    remotePatterns: [
      { protocol: "https", hostname: "lh3.googleusercontent.com" },
      { protocol: "https", hostname: "cdn.omama.vn" },
      { protocol: "https", hostname: "res.cloudinary.com" },
      { protocol: "https", hostname: "img.vietqr.io" },
      { protocol: "https", hostname: "vietqr.app" }
    ]
  }
};

export default nextConfig;
