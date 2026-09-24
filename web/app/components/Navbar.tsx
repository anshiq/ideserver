"use client";
import Link from "next/link";
import React, { useState } from "react";
import { useUserAuth } from "./UserWrapper";

export default function Navbar() {
  const [open, setOpen] = useState(false);
  const { me, logout } = useUserAuth();
  return (
    <nav className="bg-gray-800 w-full">
      <div className="container mx-auto flex justify-between items-center p-4">
        <Link href="/" className="text-white text-lg font-bold">
          VPS Platform
        </Link>
        <div className="hidden md:flex gap-4 items-center">
          {me && (
            <>
              <Link href="/instances" className="text-white">Instances</Link>
              <Link href="/instances/new" className="text-white">New</Link>
              <Link href="/keys" className="text-white">SSH keys</Link>
              {me.role === "admin" && <Link href="/admin" className="text-white">Admin</Link>}
            </>
          )}
          {!me ? (
            <a href="/api/v1/auth/github/login" className="text-white underline">Login with GitHub</a>
          ) : (
            <>
              <span className="text-gray-300 text-sm">{me.login}</span>
              <button onClick={logout} className="p-2 bg-red-600 text-white rounded-md">Logout</button>
            </>
          )}
        </div>
        <span onClick={() => setOpen(!open)} className="text-white md:hidden text-3xl cursor-pointer">≡</span>
      </div>
      {open && (
        <div className="md:hidden flex flex-col gap-2 p-4 bg-gray-800">
          {me && (
            <>
              <Link href="/instances" className="text-white">Instances</Link>
              <Link href="/keys" className="text-white">SSH keys</Link>
            </>
          )}
          {!me ? <a href="/api/v1/auth/github/login" className="text-white">Login</a>
               : <button onClick={logout} className="text-white text-left">Logout</button>}
        </div>
      )}
    </nav>
  );
}
