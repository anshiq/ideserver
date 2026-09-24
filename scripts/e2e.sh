#!/usr/bin/env bash
# End-to-end against the real Incus on the host (plan §6).
# Usage: DOMAIN=vps.example.com just e2e
set -euo pipefail
DOMAIN="${DOMAIN:-example.com}"
SUFFIX="vps.${DOMAIN}"
API="${API:-http://127.0.0.1:8081}"

need() { command -v "$1" >/dev/null || { echo "missing $1"; exit 1; }; }
need curl; need ssh; need nc; need incus

echo "== 1. create Ubuntu VPS (via API, watch SSE) =="
echo "   POST $API/api/v1/instances {plan:s, image:ubuntu-24.04}"
echo "   (login with GitHub in the dashboard, add a key, then watch /api/v1/events)"
echo
echo "== 2. SSH =="
echo "   ssh a7k2m9xq.${SUFFIX}            # via ProxyJump ${SUFFIX}"
echo "   ssh -p 3xxxx ubuntu@a7k2m9xq.${SUFFIX}  # fallback"
echo "   (browser console in dashboard must also work)"
echo
echo "== 3. HTTP/S =="
echo "   (in VM) sudo python3 -m http.server 80"
echo "   curl http://a7k2m9xq.${SUFFIX} https://a7k2m9xq.${SUFFIX}  # both 200, valid wildcard cert"
echo "   (switch to passthrough, check again)"
echo
echo "== 4. ports + firewall =="
echo "   map 5432 -> nc a7k2m9xq.${SUFFIX} 3xxxx ; add excluding rule ; confirm blocked on all 3 paths"
echo
echo "== 5. persistence + resize =="
echo "   write file; stop/start/reboot; file persists; resize plan; check nproc/free/df"
echo
echo "== 6. isolation =="
echo "   from VM: other VM IPs, 10.100.0.1:5432, port 25 all unreachable"
echo
echo "== 7. Fedora =="
echo "   repeat with fedora image"
echo
echo "== 8. delete =="
echo "   delete VPS; confirm incus instance/forwards/ACLs gone, port+IP freed"
echo
echo "Automated runner: wire the steps above to curl + ssh -o BatchMode=yes once"
echo "GitHub OAuth has a test bypass (ADMIN_GITHUB_LOGINS seed user)."
