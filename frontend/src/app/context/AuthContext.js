'use client';

import { createContext, useContext, useState, useEffect } from 'react';

const AuthContext = createContext(null);

export function AuthProvider({ children }) {
  const [user, setUser] = useState(null);
  const [sessionToken, setSessionToken] = useState(null);
  const [loading, setLoading] = useState(true);

  // On mount, read session from localStorage
  useEffect(() => {
    const storedSession = localStorage.getItem('authSession');
    if (storedSession) {
      try {
        const parsed = JSON.parse(storedSession);
        if (parsed.email && parsed.sessionToken) {
          setUser({ email: parsed.email });
          setSessionToken(parsed.sessionToken);
        }
      } catch (e) {
        localStorage.removeItem('authSession');
      }
    }
    setLoading(false);
  }, []);

  const signIn = (email, token) => {
    const session = { email, sessionToken: token };
    localStorage.setItem('authSession', JSON.stringify(session));
    setUser({ email });
    setSessionToken(token);
  };

  const signOut = () => {
    localStorage.removeItem('authSession');
    localStorage.removeItem('currentVolunteer');
    setUser(null);
    setSessionToken(null);
  };

  const isAuthenticated = !!user && !!sessionToken;

  return (
    <AuthContext.Provider value={{ user, sessionToken, isAuthenticated, loading, signIn, signOut }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
}
