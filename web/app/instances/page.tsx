"use client";
import Link from "next/link";
import { useEffect, useState } from "react";
import { apiFetch, Instance } from "@/lib/api";
import { useUserAuth } from "@/app/components/UserWrapper";

export default function InstancesPage() {
  const { me, lastEvent } = useUserAuth();
  const [items, setItems] = useState<Instance[]>([]);
  const [err, setErr] = useState<string | null>(null);

  const load = async () => {
    try {
      setItems(await apiFetch<Instance[]>("/v1/instances"));
    } catch (e) {
      setErr(e instanceof Error ? e.message : "load failed");
    }
  };
  useEffect(() => {
    if (me) load();
  }, [me, lastEvent]);

  if (!me) return <p className="p-8">Login with GitHub to manage VPSes.</p>;
  return (
    <div className="p-6 max-w-4xl mx-auto">
      <div className="flex justify-between items-center mb-4">
        <h1 className="text-2xl font-bold">Instances</h1>
        <Link href="/instances/new" className="px-3 py-2 bg-blue-600 text-white rounded">New VPS</Link>
      </div>
      {err && <p className="text-red-400">{err}</p>}
      <div className="grid gap-3">
        {items.map((i) => (
          <Link key={i.id} href={`/instances/${i.id}`} className="bg-white text-black p-4 rounded shadow flex justify-between">
            <span className="font-mono font-bold">{i.id} <span className="font-sans font-normal">· {i.name}</span></span>
            <span className="text-sm">{i.state} · {i.plan_id} · {i.image_id}</span>
          </Link>
        ))}
        {items.length === 0 && <p className="text-gray-400">No instances yet.</p>}
      </div>
    </div>
  );
}
