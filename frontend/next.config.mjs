/** @type {import('next').NextConfig} */
const nextConfig = {
  outputFileTracingRoot: import.meta.dirname,
  // Production API URLs are baked in at build time via NEXT_PUBLIC_ env vars

  async headers() {
    // Content-Security-Policy is assembled as an array of directives for
    // readability.  Notes on each directive:
    //
    //   script-src 'unsafe-inline'  — Next.js injects inline scripts for
    //     hydration (__NEXT_DATA__). Remove if you add nonce-based CSP later.
    //
    //   connect-src https: http://localhost:*  — GraphQL API calls go to the
    //     backend, which may be on a different origin (Railway URL in prod,
    //     localhost:8080 in dev). 'self' alone would block them.
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
      "connect-src 'self' https: http://localhost:*",
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

