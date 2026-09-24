"use client";
import { useEffect, useState } from "react";
import { apiFetch, SshKey } from "@/lib/api";

export default function KeysPage() {
  const [keys, setKeys] = useState<SshKey[]>([]);
  const [name, setName] = useState("");
  const [pub, setPub] = useState("");
  const load = async () => setKeys(await apiFetch<SshKey[]>("/v1/ssh-keys").catch(() => []));
  useEffect(() => { load(); }, []);
  return (
    <div className="p-6 max-w-2xl mx-auto flex flex-col gap-4">
      <h1 className="text-2xl font-bold">SSH keys</h1>
      <button onClick={async () => { await apiFetch("/v1/ssh-keys/import-github", { method: "POST" }); load(); }} className="px-3 py-1 bg-blue-600 text-white rounded w-fit">Import from github.com/{"{login}"}.keys</button>
      {keys.map((k) => (
        <div key={k.id} className="bg-white text-black p-3 rounded flex justify-between">
          <span>{k.name} <code className="text-xs">{k.fingerprint}</code></span>
          <button onClick={async () => { await apiFetch(`/v1/ssh-keys/${k.id}`, { method: "DELETE" }); load(); }} className="text-red-600">delete</button>
        </div>
      ))}
      <form onSubmit={async (e) => { e.preventDefault(); await apiFetch("/v1/ssh-keys", { method: "POST", body: JSON.stringify({ name, publicKey: pub }) }); setName(""); setPub(""); load(); }} className="flex flex-col gap-2">
        <input value={name} onChange={(e) => setName(e.target.value)} placeholder="name" className="p-2 border rounded text-black" required />
        <textarea value={pub} onChange={(e) => setPub(e.target.value)} placeholder="ssh-ed25519 AAAA…" className="p-2 border rounded text-black" required />
        <button className="px-3 py-1 bg-green-600 text-white rounded">Add key</button>
      </form>
    </div>
  );
}
