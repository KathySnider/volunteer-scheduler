'use client';

import { useState } from 'react';
import { Mail, ArrowLeft } from 'lucide-react';
import Link from 'next/link';

export default function SignInPage() {
  const [email, setEmail] = useState('');
  const [loading, setLoading] = useState(false);
  const [emailSent, setEmailSent] = useState(false);
  const [error, setError] = useState('');

  const handleSubmit = async (e) => {
    e.preventDefault();
    setLoading(true);
    setError('');

    const mutation = `
      mutation($email: String!) {
        requestMagicLink(email: $email) {
          success
          message
          email
        }
      }
    `;

    try {
      const response = await fetch('http://localhost:8080/query', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ query: mutation, variables: { email } })
      });

      const result = await response.json();

      if (result.errors) {
        setError(result.errors[0].message);
      } else if (result.data?.requestMagicLink?.success) {
        setEmailSent(true);
      } else {
        setError(result.data?.requestMagicLink?.message || 'Failed to send magic link');
      }
    } catch (err) {
      setError('Failed to connect to server. Make sure the backend is running.');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50">
      <div className="bg-white p-8 rounded-lg shadow-md max-w-md w-full mx-4">
        <h1 className="text-3xl font-bold mb-2 text-blue-800">Sign In</h1>
        <p className="text-gray-500 mb-8">AARP Volunteer System</p>

        {error && (
          <div className="p-3 bg-red-50 text-red-800 rounded-lg mb-4 text-sm">
            {error}
          </div>
        )}

        {!emailSent ? (
          <form onSubmit={handleSubmit}>
            <label className="block mb-2 text-sm font-semibold text-gray-700">
              Email Address
            </label>
            <input
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              required
              placeholder="your.email@example.com"
              className="w-full px-4 py-3 border-2 border-gray-200 rounded-lg text-base mb-4 focus:outline-none focus:ring-2 focus:ring-blue-500 bg-white"
            />
            <button
              type="submit"
              disabled={loading}
              className="w-full py-3 bg-blue-800 text-white rounded-lg font-semibold hover:bg-blue-900 transition-colors disabled:opacity-70 disabled:cursor-not-allowed"
            >
              {loading ? 'Sending...' : 'Sign in with Email'}
            </button>
          </form>
        ) : (
          <div className="p-6 bg-green-50 rounded-lg text-center">
            <Mail className="w-12 h-12 text-green-600 mx-auto mb-4" />
            <h3 className="text-lg font-semibold text-green-800 mb-2">Check your email!</h3>
            <p className="text-green-700 mb-4">
              We sent a magic link to <strong>{email}</strong>
            </p>
            <button
              onClick={() => setEmailSent(false)}
              className="mt-2 px-4 py-2 bg-white border-2 border-green-600 text-green-600 rounded-lg font-semibold hover:bg-green-50 transition-colors"
            >
              Send another email
            </button>
          </div>
        )}

        <div className="mt-8 text-center">
          <Link href="/" className="text-blue-600 hover:text-blue-700 text-sm inline-flex items-center gap-1">
            <ArrowLeft className="w-4 h-4" />
            Back to home
          </Link>
        </div>
      </div>
    </div>
  );
}
