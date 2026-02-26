'use client';

import { AuthProvider } from '../context/AuthContext';

export default function Providers({ children }) {
  return <AuthProvider>{children}</AuthProvider>;
}
