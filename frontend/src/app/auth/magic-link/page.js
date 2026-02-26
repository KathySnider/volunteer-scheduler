'use client';

import { useState, useEffect, Suspense } from 'react';
import { useSearchParams, useRouter } from 'next/navigation';
import { CheckCircle, XCircle } from 'lucide-react';
import { useAuth } from '../../context/AuthContext';
import Link from 'next/link';

function MagicLinkHandler() {
  const searchParams = useSearchParams();
  const router = useRouter();
  const { signIn } = useAuth();
  const [status, setStatus] = useState('loading');
  const [message, setMessage] = useState('');

  useEffect(() => {
    const token = searchParams.get('token');
    if (!token) {
      setStatus('error');
      setMessage('No token provided. Please request a new magic link.');
      return;
    }

    consumeToken(token);
  }, [searchParams]);

  const consumeToken = async (token) => {
    const mutation = `
      mutation($token: String!) {
        consumeMagicLink(token: $token) {
          success
          message
          email
          sessionToken
        }
      }
    `;

    try {
      const response = await fetch('http://localhost:8080/query', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ query: mutation, variables: { token } })
      });

      const result = await response.json();

      if (result.errors) {
        setStatus('error');
        setMessage(result.errors[0].message);
      } else if (result.data?.consumeMagicLink?.success) {
        const { email, sessionToken } = result.data.consumeMagicLink;
        signIn(email, sessionToken);
        setStatus('success');
        setTimeout(() => router.push('/'), 1500);
      } else {
        setStatus('error');
        setMessage(result.data?.consumeMagicLink?.message || 'Invalid or expired magic link');
      }
    } catch (err) {
      setStatus('error');
      setMessage('Failed to connect to server. Please try again.');
    }
  };

  if (status === 'loading') {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-50">
        <div className="text-center">
          <div className="inline-block animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
          <p className="mt-4 text-gray-600">Signing you in...</p>
        </div>
      </div>
    );
  }

  if (status === 'success') {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-50">
        <div className="bg-white p-8 rounded-lg shadow-md max-w-md w-full mx-4 text-center">
          <CheckCircle className="w-16 h-16 text-green-600 mx-auto mb-4" />
          <h2 className="text-2xl font-bold text-green-800 mb-2">You're signed in!</h2>
          <p className="text-gray-500 mb-6">Redirecting to the events page...</p>
          <Link
            href="/"
            className="inline-block px-6 py-3 bg-blue-800 text-white rounded-lg font-semibold hover:bg-blue-900 transition-colors"
          >
            Go to Events
          </Link>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50">
      <div className="bg-white p-8 rounded-lg shadow-md max-w-md w-full mx-4 text-center">
        <XCircle className="w-16 h-16 text-red-500 mx-auto mb-4" />
        <h2 className="text-2xl font-bold text-red-800 mb-2">Sign-in Failed</h2>
        <p className="text-gray-600 mb-6">{message}</p>
        <Link
          href="/auth/signin"
          className="inline-block px-6 py-3 bg-blue-800 text-white rounded-lg font-semibold hover:bg-blue-900 transition-colors"
        >
          Try Again
        </Link>
      </div>
    </div>
  );
}

export default function MagicLinkPage() {
  return (
    <Suspense fallback={
      <div className="min-h-screen flex items-center justify-center bg-gray-50">
        <div className="inline-block animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
      </div>
    }>
      <MagicLinkHandler />
    </Suspense>
  );
}
