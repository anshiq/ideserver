"use client";
import { useEffect, useState } from "react";
import { apiFetch, AdminUser, HostCapacity } from "@/lib/api";

export default function AdminPage() {
  const [users, setUsers] = useState<AdminUser[]>([]);
  const [host, setHost] = useState<HostCapacity | null>(null);
  useEffect(() => {
    apiFetch<AdminUser[]>("/v1/admin/users").then(setUsers).catch(() => {});
    apiFetch<HostCapacity>("/v1/admin/host").then(setHost).catch(() => {});
  }, []);
  return (
    <div className="p-6 max-w-4xl mx-auto flex flex-col gap-4">
      <h1 className="text-2xl font-bold">Admin</h1>
      <section className="bg-white text-black p-4 rounded">
        <h2 className="font-bold">Host capacity</h2>
        <pre className="text-sm">{JSON.stringify(host, null, 2) || "—"}</pre>
      </section>
      <section className="bg-white text-black p-4 rounded">
        <h2 className="font-bold">Users (approve / quota / suspend)</h2>
        {users.map((u) => (
          <div key={u.id} className="flex gap-2 items-center border-b py-1">
            <span>{u.login} · {u.status} · {u.role}</span>
          </div>
        ))}
      </section>
    </div>
  );
}
