/** @type {import('next').NextConfig} */
const nextConfig = {
  outputFileTracingRoot: import.meta.dirname,

  // ---------------------------------------------------------------------------
  // GraphQL proxy rewrites
  //
  // All /graphql/* requests from the browser are proxied through this Next.js
  // server to the backend.  This keeps cookies on a single domain so that:
  //   - The session cookie set after login is readable by Next.js middleware
  //     (which runs on the frontend server, not the backend).
  //   - There is no cross-origin cookie mismatch between Railway services.
  //
  // BACKEND_INTERNAL_URL is a server-side-only env var (no NEXT_PUBLIC_ prefix):
  //   Docker Compose : http://api:8080
  //   Railway        : http://<backend-service>.railway.internal:8080
  //   Local bare dev : http://localhost:8080  (default)
  // ---------------------------------------------------------------------------
  async rewrites() {
    const backendUrl = process.env.BACKEND_INTERNAL_URL || 'http://localhost:8080'
    return [
      {
        source: '/graphql/:path*',
        destination: `${backendUrl}/graphql/:path*`,
      },
    ]
  },

  async headers() {
    // Content-Security-Policy is assembled as an array of directives for
    // readability.  Notes on each directive:
    //
    //   script-src 'unsafe-inline'  — Next.js injects inline scripts for
    //     hydration (__NEXT_DATA__). Remove if you add nonce-based CSP later.
    //
    //   connect-src 'self'  — GraphQL calls now go to /graphql/* on this
    //     same origin (proxied to the backend via rewrites above), so
    //     'self' is sufficient. http://localhost:* kept for bare local dev.
    //
    //   frame-ancestors 'none'  — prevents this app from being embedded in
    //     an iframe on any other site (clickjacking defence). Redundant with
    //     X-Frame-Options: DENY but covers CSP-aware browsers that may ignore
    //     the older header.
    //
    //   Strict-Transport-Security  — browsers ignore this over plain HTTP so
    //     it is safe to send in development; it takes effect in production.
    const csp = [
      "default-src 'self'",
      "script-src 'self' 'unsafe-inline'",
      "style-src 'self' 'unsafe-inline'",
      "img-src 'self' data: blob:",
      "font-src 'self'",
      "connect-src 'self' http://localhost:*",
      "frame-ancestors 'none'",
      "base-uri 'self'",
      "form-action 'self'",
    ].join('; ')

    return [
      {
        source: '/(.*)',
        headers: [
          { key: 'X-Frame-Options',        value: 'DENY' },
          { key: 'X-Content-Type-Options',  value: 'nosniff' },
          { key: 'Referrer-Policy',         value: 'strict-origin-when-cross-origin' },
          { key: 'Permissions-Policy',      value: 'camera=(), microphone=(), geolocation=()' },
          { key: 'Strict-Transport-Security', value: 'max-age=63072000; includeSubDomains; preload' },
          { key: 'Content-Security-Policy', value: csp },
        ],
      },
    ]
  },
}

export default nextConfig

