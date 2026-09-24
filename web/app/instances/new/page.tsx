"use client";
import { SubmitEvent, useEffect, useState } from "react";
import { apiFetch, Image, Instance, Plan, SshKey } from "@/lib/api";

export default function NewInstancePage() {
  const [plans, setPlans] = useState<Plan[]>([]);
  const [images, setImages] = useState<Image[]>([]);
  const [keys, setKeys] = useState<SshKey[]>([]);
  const [form, setForm] = useState({ name: "", plan: "s", image: "ubuntu-24.04", userData: "", sshKeys: [] as number[] });
  const [msg, setMsg] = useState("");

  useEffect(() => {
    apiFetch<Plan[]>("/v1/plans").then(setPlans).catch(() => {});
    apiFetch<Image[]>("/v1/images").then(setImages).catch(() => {});
    apiFetch<SshKey[]>("/v1/ssh-keys").then(setKeys).catch(() => {});
  }, []);

  const submit = async (e: SubmitEvent) => {
    e.preventDefault();
    setMsg("creating…");
    try {
      const out = await apiFetch<Instance>("/v1/instances", { method: "POST", body: JSON.stringify(form) });
      window.location.replace(`/instances/${out.id}`);
    } catch (err) {
      setMsg(err instanceof Error ? err.message : "create failed");
    }
  };

  return (
    <form onSubmit={submit} className="p-6 max-w-xl mx-auto flex flex-col gap-3">
      <h1 className="text-2xl font-bold">New VPS</h1>
      <label>Name<input value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} className="w-full p-2 border rounded text-black" required maxLength={64} /></label>
      <label>Plan<select value={form.plan} onChange={(e) => setForm({ ...form, plan: e.target.value })} className="w-full p-2 border rounded text-black">
        {plans.map((p) => <option key={p.slug} value={p.slug}>{p.slug} — {p.vcpu}vCPU {p.memory_mb}MB {p.disk_gb}GB</option>)}
      </select></label>
      <label>OS image<select value={form.image} onChange={(e) => setForm({ ...form, image: e.target.value })} className="w-full p-2 border rounded text-black">
        {images.map((i) => <option key={i.slug} value={i.slug}>{i.distro} {i.version}</option>)}
      </select></label>
      <fieldset>SSH keys{keys.map((k) => (
        <label key={k.id} className="flex gap-2"><input type="checkbox" checked={form.sshKeys.includes(k.id)}
          onChange={(e) => setForm({ ...form, sshKeys: e.target.checked ? [...form.sshKeys, k.id] : form.sshKeys.filter((x) => x !== k.id) })} />{k.name}</label>
      ))}</fieldset>
      <label>user-data (optional)<textarea value={form.userData} onChange={(e) => setForm({ ...form, userData: e.target.value })} className="w-full p-2 border rounded text-black" rows={4} /></label>
      <button className="px-4 py-2 bg-green-600 text-white rounded">Create</button>
      {msg && <p>{msg}</p>}
    </form>
  );
}
