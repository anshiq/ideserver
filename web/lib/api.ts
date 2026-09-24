// Typed API client: same-origin /api (no CORS, plan §1.2/§1.6).
// Generated types (lib/api.d.ts) come from the huma OpenAPI via
// `just gen` (openapi-typescript). This file is the hand-written wrapper.
import createClient from "openapi-fetch";
import type { components, paths } from "./api.d";

export const api = createClient<paths>({
  baseUrl: "/api",
  headers: { "X-Requested-With": "fetch" },
  credentials: "same-origin",
});

// Aliases onto the generated schemas (lib/api.d.ts, from `just gen` /
// huma's OpenAPI — plan §1.8) so pages get real types instead of `any`.
export type Instance = components["schemas"]["InstanceOut"];
export type Plan = components["schemas"]["Plans"];
export type Image = components["schemas"]["Images"];
export type SshKey = components["schemas"]["SshKeys"];
export type FirewallRule = components["schemas"]["FirewallRules"];
export type PortMapping = components["schemas"]["PortMappings"];
export type AdminUser = components["schemas"]["Users"];
export type HostCapacity = components["schemas"]["HostCapacity"];
export type InstanceStats = components["schemas"]["StatsOutputBody"];

export async function apiFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`/api${path}`, {
    credentials: "same-origin",
    headers: { "X-Requested-With": "fetch", "Content-Type": "application/json", ...(init?.headers || {}) },
    ...init,
  });
  if (!res.ok) throw new Error(`${init?.method || "GET"} ${path}: ${res.status}`);
  if (res.status === 204) return undefined as T;
  return res.json() as Promise<T>;
}

// SSE hook: subscribes to /api/v1/events (NOTIFY instance_events fan-out).
import { useEffect, useState } from "react";

export function useInstanceEvents(onEvent: (msg: string) => void) {
  useEffect(() => {
    const src = new EventSource("/api/v1/events");
    src.onmessage = (e) => onEvent(e.data);
    return () => src.close();
  }, [onEvent]);
}

export function useNow(intervalMs = 5000) {
  const [now, setNow] = useState(Date.now());
  useEffect(() => {
    const t = setInterval(() => setNow(Date.now()), intervalMs);
    return () => clearInterval(t);
  }, [intervalMs]);
  return now;
}
