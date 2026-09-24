"use client";
import React, { createContext, useContext, useEffect, useState, ReactNode, useCallback } from "react";

type Me = { id: number; login: string; role: string; status: string } | null;

type AuthContextType = {
  me: Me;
  loading: boolean;
  refresh: () => void;
  logout: () => void;
  lastEvent: string | null;
};

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export function UserAuthWrapper({ children }: { children: ReactNode }) {
  const [me, setMe] = useState<Me>(null);
  const [loading, setLoading] = useState(true);
  const [lastEvent, setLastEvent] = useState<string | null>(null);

  const refresh = useCallback(async () => {
    try {
      const res = await fetch("/api/v1/me", { credentials: "same-origin" });
      if (res.ok) setMe(await res.json());
      else setMe(null);
    } catch {
      setMe(null);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    refresh();
  }, [refresh]);

  // SSE: session cookie authenticates the stream; every instance state
  // change revalidates (plan §1.5 step 3).
  useEffect(() => {
    if (!me) return;
    const src = new EventSource("/api/v1/events");
    src.onmessage = (e) => setLastEvent(e.data);
    return () => src.close();
  }, [me]);

  const logout = useCallback(async () => {
    await fetch("/api/v1/auth/logout", {
      method: "POST",
      credentials: "same-origin",
      headers: { "X-Requested-With": "fetch" },
    });
    setMe(null);
    window.location.replace("/");
  }, []);

  if (loading) return <>Loading…</>;
  return (
    <AuthContext.Provider value={{ me, loading, refresh, logout, lastEvent }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useUserAuth(): AuthContextType {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useUserAuth must be used within UserAuthWrapper");
  return ctx;
}
