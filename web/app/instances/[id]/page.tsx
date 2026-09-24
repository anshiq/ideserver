"use client";
import { useCallback, useEffect, useRef, useState } from "react";
import { useParams } from "next/navigation";
import { apiFetch, FirewallRule, Instance, InstanceStats, PortMapping } from "@/lib/api";
import { useUserAuth } from "@/app/components/UserWrapper";
import "@xterm/xterm/css/xterm.css";

export default function InstanceDetail() {
  const { id } = useParams() as { id: string };
  const { lastEvent } = useUserAuth();
  const [inst, setInst] = useState<Instance | null>(null);
  const [sshConfig, setSshConfig] = useState("");
  const [stats, setStats] = useState<InstanceStats | null>(null);
  const [fw, setFw] = useState<FirewallRule[]>([]);
  const [ports, setPorts] = useState<PortMapping[]>([]);
  const termRef = useRef<HTMLDivElement>(null);

  const load = useCallback(async () => {
    setInst(await apiFetch<Instance>(`/v1/instances/${id}`).catch(() => null));
    setSshConfig(await apiFetch<{ config: string }>(`/v1/instances/${id}/ssh-config`).then((r) => r.config).catch(() => ""));
    setStats(await apiFetch<InstanceStats>(`/v1/instances/${id}/stats`).catch(() => null));
    setFw(await apiFetch<FirewallRule[]>(`/v1/instances/${id}/firewall`).catch(() => []));
    setPorts(await apiFetch<PortMapping[]>(`/v1/instances/${id}/ports`).catch(() => []));
  }, [id]);
  useEffect(() => {
    load();
  }, [load, lastEvent]);

  // Browser console: xterm → WS /api/v1/instances/{id}/console (hostd relay).
  // xterm touches browser globals (`self`) at module-import time, which
  // crashes Next's server render of this route, so it's loaded dynamically
  // here rather than as a static import — this effect only ever runs in
  // the browser anyway.
  const running = inst?.state === "running";
  useEffect(() => {
    if (!termRef.current || !running) return;
    let cancelled = false;
    let cleanup = () => {};
    (async () => {
      const [{ Terminal }, { FitAddon }] = await Promise.all([
        import("@xterm/xterm"),
        import("@xterm/addon-fit"),
      ]);
      if (cancelled || !termRef.current) return;
      const term = new Terminal({ convertEol: true });
      const fit = new FitAddon();
      term.loadAddon(fit);
      term.open(termRef.current);
      fit.fit();
      const ws = new WebSocket(`${location.protocol === "https:" ? "wss" : "ws"}://${location.host}/api/v1/instances/${id}/console`);
      ws.binaryType = "arraybuffer";
      term.onData((d) => ws.readyState === 1 && ws.send(d));
      ws.onmessage = (e) => term.write(typeof e.data === "string" ? e.data : new Uint8Array(e.data));
      cleanup = () => { ws.close(); term.dispose(); };
    })();
    return () => { cancelled = true; cleanup(); };
  }, [id, running]);

  const act = async (action: string, plan?: string) => {
    await apiFetch(`/v1/instances/${id}/actions`, { method: "POST", body: JSON.stringify(plan ? { action, plan } : { action }) });
    load();
  };

  if (!inst) return <p className="p-8">Loading…</p>;
  return (
    <div className="p-6 max-w-4xl mx-auto flex flex-col gap-4">
      <h1 className="text-2xl font-bold font-mono">{inst.id} <span className="font-sans">· {inst.name} · {inst.state}</span></h1>
      {inst.last_error && <p className="text-red-400">{inst.last_error}</p>}
      <div className="flex gap-2 flex-wrap">
        <button onClick={() => act("start")} className="px-3 py-1 bg-green-600 text-white rounded">Start</button>
        <button onClick={() => act("stop")} className="px-3 py-1 bg-yellow-600 text-white rounded">Stop</button>
        <button onClick={() => act("reboot")} className="px-3 py-1 bg-blue-600 text-white rounded">Reboot</button>
        <button onClick={() => { const p = prompt("new plan slug"); if (p) act("resize", p); }} className="px-3 py-1 bg-purple-600 text-white rounded">Resize (stopped only)</button>
        <button onClick={async () => { if (confirm("delete?")) { await apiFetch(`/v1/instances/${id}`, { method: "DELETE" }); location.replace("/instances"); } }} className="px-3 py-1 bg-red-600 text-white rounded">Delete</button>
      </div>
      <section className="bg-white text-black p-4 rounded">
        <h2 className="font-bold">SSH</h2>
        <pre className="text-sm whitespace-pre-wrap">{sshConfig}</pre>
        <p className="text-sm">fingerprints: {inst.host_key_fps || "pending (booting)"}</p>
      </section>
      <section className="bg-white text-black p-4 rounded">
        <h2 className="font-bold">Live stats</h2>
        <pre className="text-sm">{JSON.stringify(stats, null, 2) || "—"}</pre>
      </section>
      <section className="bg-white text-black p-4 rounded">
        <h2 className="font-bold">Browser console</h2>
        <div ref={termRef} className="h-64 bg-black" />
      </section>
      <section className="bg-white text-black p-4 rounded">
        <h2 className="font-bold">Firewall (inbound)</h2>
        <pre className="text-sm">{JSON.stringify(fw, null, 2)}</pre>
      </section>
      <section className="bg-white text-black p-4 rounded">
        <h2 className="font-bold">Port mappings</h2>
        <pre className="text-sm">{JSON.stringify(ports, null, 2)}</pre>
      </section>
    </div>
  );
}
