# NixOS module for the single-host VPS platform (plan §1, §2).
#
# Usage in the host configuration:
#   imports = [ ideserver.nixosModules.default ];
#   services.vps = {
#     enable = true;
#     domain = "example.com";
#     hostIPv4 = "203.0.113.10";
#     github.clientId = "...";
#     github.secretFile = "/run/secrets/github-secret";
#     sessionSecretFile = "/run/secrets/session-secret";
#     acmeEmail = "ops@example.com";
#     dnsProvider = ""; # libdns provider name for DNS-01 (plan §1.2)
#   };
{ config, lib, pkgs, ... }:

with lib;

let
  cfg = config.services.vps;
  vpsPkgs = cfg.package;
in {
  options.services.vps = {
    enable = mkEnableOption "single-host VPS platform (Incus + Go + Postgres)";

    package = mkOption {
      type = types.package;
      description = "Go binaries (cmd/api, cmd/hostd, cmd/edge, cmd/sshgw).";
    };

    webPackage = mkOption {
      type = types.nullOr types.package;
      default = null;
      description = "Next.js standalone output (web/). Null = run web from source checkout.";
    };

    domain = mkOption { type = types.str; example = "example.com"; };
    hostIPv4 = mkOption { type = types.str; example = "203.0.113.10"; description = "The single public IPv4."; };
    bridgeIPv4 = mkOption { type = types.str; default = "10.100.0.1/16"; };
    zfsPool = mkOption { type = types.str; default = "vps"; description = "ZFS pool for Incus storage (mirrored pair preferred)."; };
    zfsAutoCreate = mkOption {
      type = types.bool;
      default = false;
      description = ''
        false (default, for a real deployment): Incus imports an existing
        zpool named `zfsPool` — create it yourself first, e.g.
        `zpool create vps mirror /dev/sda /dev/sdb`, so you control the
        disks and redundancy. Incus's zfs driver otherwise reports "cannot
        import: no such pool available" and vps-hostd crash-loops.
        true (quick test/single-disk hosts): Incus creates a new
        loop-file-backed zpool automatically (`size` defaults to 20% of
        free disk, 5-30 GiB) — convenient, but the loop file lives on
        whatever filesystem backs it, so it is not real redundancy.
      '';
    };

    github.clientId = mkOption { type = types.str; };
    github.secretFile = mkOption { type = types.path; };
    sessionSecretFile = mkOption { type = types.path; };
    acmeEmail = mkOption { type = types.str; default = ""; };
    dnsProvider = mkOption { type = types.str; default = ""; description = "libdns provider for DNS-01 wildcard certs."; };

    extraHttpPorts = mkOption { type = types.listOf types.port; default = [ 3000 5000 8000 8080 ]; };
  };

  config = mkIf cfg.enable (let
    appDomain = "app.${cfg.domain}";
    vpsSuffix = "vps.${cfg.domain}";
    dbUrl = "postgres://vps:vps@/vps?host=/run/postgresql&sslmode=disable";
  in {
    assertions = [
      { assertion = cfg.hostIPv4 != ""; message = "services.vps.hostIPv4 (the single public IP) is required."; }
    ];

    # --- Host sshd moves to 2222 key-only BEFORE sshgw binds 22 (plan §1.2) ---
    services.openssh = {
      enable = true;
      ports = [ 2222 ];
      settings = {
        PasswordAuthentication = false;
        PermitRootLogin = "no";
      };
    };

    # --- ZFS + Incus (LXC containers, ADR-0005 — no /dev/kvm required) ---
    boot.supportedFilesystems = [ "zfs" ];
    networking.hostId = mkDefault "8425e349"; # required for ZFS; override per host

    virtualisation.incus = {
      enable = true;
      preseed = {
        networks = [{
          name = "vpsbr0";
          type = "bridge";
          config = {
            "ipv4.address" = cfg.bridgeIPv4;
            "ipv4.nat" = "true";
            "ipv6.address" = "none";
          };
        }];
        storage_pools = [{
          name = cfg.zfsPool;
          driver = "zfs";
          # source unset => Incus creates a new loop-backed pool named
          # zfs.pool_name; source = an existing pool name => Incus imports
          # it (fails loudly if that zpool doesn't already exist).
          config = if cfg.zfsAutoCreate
            then { "zfs.pool_name" = cfg.zfsPool; }
            else { source = cfg.zfsPool; };
        }];
        profiles = [{
          name = "default";
          devices = {
            root = { type = "disk"; pool = cfg.zfsPool; path = "/"; };
            eth0 = { type = "nic"; network = "vpsbr0"; };
          };
        }];
        projects = [{ name = "vps"; description = "VPS tenant project"; }];
      };
    };
    systemd.services.incus.serviceConfig.SupplementaryGroups = [ "incus-admin" ];

    # --- PostgreSQL 17 (only database, plan §1.5/§1.7) ---
    # Each Go binary runs as its own system user (vps-api, vps-hostd,
    # vps-sshgw — see below) but all connect as the single `vps` database
    # role over the local unix socket. Plain `peer` auth requires the
    # system and database usernames to match, which they don't, so this
    # maps every service user onto the `vps` role explicitly; without it
    # every service fails to start with "Peer authentication failed".
    services.postgresql = {
      enable = true;
      package = pkgs.postgresql_17;
      ensureDatabases = [ "vps" ];
      ensureUsers = [{
        name = "vps";
        ensureDBOwnership = true;
      }];
      identMap = ''
        vpsmap vps-api    vps
        vpsmap vps-hostd  vps
        vpsmap vps-sshgw  vps
        vpsmap vps-edge   vps
        vpsmap vps-web    vps
        vpsmap root       vps
      '';
      authentication = lib.mkBefore ''
        local vps vps peer map=vpsmap
      '';
    };

    # --- Firewall: 22 (sshgw), 80/443 (+ extra HTTP), 30000-39999 mapped ---
    networking.nftables.enable = true;
    networking.firewall = {
      enable = true;
      allowedTCPPorts = [ 22 80 443 2222 ] ++ cfg.extraHttpPorts;
      allowedTCPPortRanges = [{ from = 30000; to = 39999; }];
      allowedUDPPortRanges = [{ from = 30000; to = 39999; }];
    };

    users.groups.vps = {};
    # A separate group scoped to only api+hostd, so the hostd console-relay
    # socket (any local reader can request a PTY into a VM by id — the
    # ownership check lives in api's handleConsole, not on the socket) is
    # unreachable from vps-edge/vps-sshgw/vps-web even though they share the
    # broader "vps" group.
    users.groups.vps-console = {};
    users.users.vps-api = { isSystemUser = true; group = "vps"; extraGroups = [ "vps-console" ]; };
    users.users.vps-edge = { isSystemUser = true; group = "vps"; };
    users.users.vps-sshgw = { isSystemUser = true; group = "vps"; };
    users.users.vps-hostd = { isSystemUser = true; group = "vps-console"; extraGroups = [ "incus-admin" ]; };
    users.users.vps-web = { isSystemUser = true; group = "vps"; };

    # --- systemd units (one Go module, four binaries, plan §1.1) ---
    systemd.services.vps-api = {
      description = "VPS API (Go, loopback :8081)";
      wantedBy = [ "multi-user.target" ];
      # "postgresql.service" alone only waits for the server to start, not
      # for ensureUsers/ensureDatabases (a separate unit) to actually
      # create the `vps` role/database — without this every DB-touching
      # service races it and fails once with "role vps does not exist".
      after = [ "postgresql.service" "postgresql-setup-start.service" ];
      environment = {
        DATABASE_URL = dbUrl;
        DOMAIN = cfg.domain;
        API_LISTEN = "127.0.0.1:8081";
        SIGNUP_MODE = "approval";
        GITHUB_CLIENT_ID = cfg.github.clientId;
      };
      serviceConfig = {
        User = "vps-api";
        ExecStart = "${vpsPkgs}/bin/api";
        EnvironmentFile = [ cfg.github.secretFile cfg.sessionSecretFile ];
        Restart = "on-failure";
        NoNewPrivileges = true;
      };
    };

    systemd.services.vps-hostd = {
      description = "VPS host daemon (Incus + River workers)";
      wantedBy = [ "multi-user.target" ];
      after = [ "incus.service" "postgresql.service" "postgresql-setup-start.service" ];
      environment = {
        DATABASE_URL = dbUrl;
        DOMAIN = cfg.domain;
        HOSTD_SOCKET = "/run/vps/hostd.sock";
        INCUS_PROJECT = "vps";
        BRIDGE_NAME = "vpsbr0";
      };
      serviceConfig = {
        User = "vps-hostd";
        Group = "vps-console";
        RuntimeDirectory = "vps";
        RuntimeDirectoryMode = "0750";
        ExecStart = "${vpsPkgs}/bin/hostd";
        Restart = "on-failure";
      };
    };

    systemd.services.vps-edge = {
      description = "VPS edge proxy (public :80/:443)";
      wantedBy = [ "multi-user.target" ];
      # edge queries Postgres directly for its route cache, independent of
      # vps-api, so it needs the same postgresql-setup-start ordering.
      after = [ "network.target" "vps-api.service" "postgresql.service" "postgresql-setup-start.service" ];
      environment = {
        DATABASE_URL = dbUrl;
        DOMAIN = cfg.domain;
        EDGE_HTTP = ":80";
        EDGE_HTTPS = ":443";
        WEB_ADDR = "127.0.0.1:13000";
        API_ADDR = "127.0.0.1:8081";
        CERT_DIR = "/var/lib/vps/certs";
        ACME_EMAIL = cfg.acmeEmail;
        DNS_PROVIDER = cfg.dnsProvider;
      };
      serviceConfig = {
        User = "vps-edge";
        ExecStart = "${vpsPkgs}/bin/edge";
        AmbientCapabilities = [ "CAP_NET_BIND_SERVICE" ];
        CapabilityBoundingSet = [ "CAP_NET_BIND_SERVICE" ];
        Restart = "on-failure";
        NoNewPrivileges = true;
      };
    };

    systemd.services.vps-sshgw = {
      description = "VPS SSH jump gateway (public :22)";
      wantedBy = [ "multi-user.target" ];
      after = [ "network.target" "postgresql.service" "postgresql-setup-start.service" ];
      environment = {
        DATABASE_URL = dbUrl;
        DOMAIN = cfg.domain;
        SSHGW_LISTEN = ":22";
        SSHGW_HOST_KEY = "/var/lib/vps/sshgw_hostkey";
      };
      serviceConfig = {
        User = "vps-sshgw";
        ExecStart = "${vpsPkgs}/bin/sshgw";
        AmbientCapabilities = [ "CAP_NET_BIND_SERVICE" ];
        CapabilityBoundingSet = [ "CAP_NET_BIND_SERVICE" ];
        Restart = "on-failure";
        NoNewPrivileges = true;
        StateDirectory = "vps";
      };
    };

    systemd.services.vps-web = {
      description = "VPS dashboard (Next.js standalone :13000)";
      wantedBy = [ "multi-user.target" ];
      after = [ "vps-api.service" ];
      environment = {
        PORT = "13000";
        HOSTNAME = "127.0.0.1";
        NEXT_PUBLIC_APP_DOMAIN = appDomain;
        NEXT_PUBLIC_VPS_SUFFIX = vpsSuffix;
      };
      serviceConfig = {
        User = "vps-web";
        ExecStart = mkIf (cfg.webPackage != null) "${cfg.webPackage}/bin/vps-web";
        Restart = "on-failure";
        NoNewPrivileges = true;
      };
    };
  });
}
