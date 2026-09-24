import Link from "next/link";

export default function Home() {
  return (
    <div className="p-8 max-w-3xl mx-auto text-center">
      <h1 className="text-4xl font-bold mb-4">Persistent Linux machines</h1>
      <p className="text-gray-300 mb-6">
        Ubuntu/Fedora instances that survive reboots, on <code>{"{id}.vps.{domain}"}</code> with SSH + any port.
      </p>
      <div className="flex gap-4 justify-center">
        <a href="/api/v1/auth/github/login" className="px-4 py-2 bg-white text-black rounded">Login with GitHub</a>
        <Link href="/instances" className="px-4 py-2 bg-blue-600 text-white rounded">My instances</Link>
      </div>
    </div>
  );
}
