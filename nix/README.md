# Smallworld on NixOS

This flake packages the gateway, room workers, map generator, and React UI. Its
NixOS module also configures nginx, PostgreSQL, Redis, service users, startup
ordering, persistence, firewall access, and systemd hardening.

## Fast local/LAN deployment

Add this repository to the target machine's flake inputs:

```nix
{
  inputs.smallworld.url = "github:Arthur158/smallworld";

  outputs = { nixpkgs, smallworld, ... }@inputs: {
    nixosConfigurations.game-server = nixpkgs.lib.nixosSystem {
      system = "x86_64-linux";
      specialArgs = { inherit inputs; };
      modules = [
        smallworld.nixosModules.default
        ./configuration.nix
        {
          services.smallworld.enable = true;
        }
      ];
    };
  };
}
```

Deploy it:

```console
sudo nixos-rebuild switch --flake .#game-server
```

The default deployment listens on port 3000, opens that port in the firewall,
starts two workers, and provisions local PostgreSQL and Redis instances. Open
`http://SERVER_IP:3000/`.

## Public HTTPS deployment

Use a DNS name that already points at the machine:

```nix
services.smallworld = {
  enable = true;
  hostName = "smallworld.example.com";
  port = 443;
  enableACME = true;
  allowedOrigins = [ "https://smallworld.example.com" ];
};

security.acme = {
  acceptTerms = true;
  defaults.email = "admin@example.com";
};
```

See `example-configuration.nix` for the same configuration in a complete
module fragment.

## Module options

All options are under `services.smallworld`:

| Option | Type | Default | Meaning |
| --- | --- | --- | --- |
| `enable` | boolean | `false` | Enable the complete Smallworld stack. |
| `backendPackage` | package | the flake's `backend` package | Gateway and room-worker binaries. Override this to deploy a custom backend derivation. |
| `frontendPackage` | package | the flake's `frontend` package | Static frontend served by nginx. Override this to deploy a custom frontend derivation. |
| `mapGeneratorPackage` | package | the flake's `map-generator` package | Map-generator server package. Override this to deploy a custom map generator derivation. |
| `hostName` | null or string | `null` | Public hostname used by nginx. `null` creates a catch-all default virtual host. Required when `enableACME = true`. |
| `listenAddress` | string | `"0.0.0.0"` | Address on which nginx accepts public connections. Use `"127.0.0.1"` when another local reverse proxy is in front. |
| `port` | TCP port | `3000` | Public HTTP or HTTPS port exposed by nginx. |
| `openFirewall` | boolean | `true` | Add `port` to `networking.firewall.allowedTCPPorts`. |
| `enableACME` | boolean | `false` | Obtain an ACME certificate for `hostName`, enable TLS, and redirect HTTP to HTTPS. The host must also configure `security.acme.acceptTerms` and an ACME email address. |
| `workerCount` | positive integer | `2` | Number of room-worker systemd services and the shared `ROOM_WORKER_COUNT`. Do not change it while rooms are active because it changes room-to-worker assignment. |
| `workerHealthPortBase` | TCP port | `8100` | First loopback health port for workers. Worker `N` listens on `workerHealthPortBase + N`; the complete range must not exceed `65535`. |
| `mapGeneratorPort` | TCP port | `3001` | Loopback port for the map-generator API. |
| `allowedOrigins` | list of strings | `[ "*" ]` | WebSocket origins accepted by the gateway. For a public deployment, set this to the exact frontend origin. |
| `environmentFile` | null or path | `null` | Optional systemd environment file applied to the gateway, workers, and map generator. Use it for secrets and external-service overrides rather than putting credentials in the Nix store. |
| `database.createLocally` | boolean | `true` | Provision a local PostgreSQL database and `smallworld` role using peer authentication. |
| `database.url` | string | `""` | External `DATABASE_URL` used when `database.createLocally = false`. It or `environmentFile` must be set in that case. |
| `redis.createLocally` | boolean | `true` | Provision a persistent Redis instance bound only to loopback. |
| `redis.address` | string | `""` | External `REDIS_ADDR` used when `redis.createLocally = false`. It or `environmentFile` must be set in that case. |
| `redis.port` | TCP port | `6379` | Port of the locally provisioned Redis instance. Ignored when `redis.createLocally = false`. |

For example, a small single-machine deployment with one room worker is:

```nix
services.smallworld = {
  enable = true;
  workerCount = 1;
  allowedOrigins = [ "http://smallworld.lan:3000" ];
};
```

## Operations

```console
systemctl status smallworld-gateway smallworld-worker-0 smallworld-worker-1
journalctl -u 'smallworld-*' -f
curl --fail http://127.0.0.1:8080/readyz
curl --fail http://127.0.0.1:8100/readyz
curl --fail http://127.0.0.1:3001/api/health
```

PostgreSQL data lives under `/var/lib/postgresql`; the named Redis instance
uses `/var/lib/redis-smallworld`. Both survive rebuilds and reboots.

`workerCount` controls both the number of systemd room-worker services and the
application's `ROOM_WORKER_COUNT`. Do not change it while rooms are active:
room ownership is calculated from this value.

For external PostgreSQL or Redis, disable `createLocally` and set `url` or
`address`. Put credentials in `environmentFile`, not directly in Nix code,
because evaluated Nix strings are stored world-readable in the Nix store:

```nix
services.smallworld = {
  database.createLocally = false;
  redis.createLocally = false;
  environmentFile = "/run/keys/smallworld.env";
};
```

The environment file can contain `DATABASE_URL=...`, `REDIS_ADDR=...`, or
`REDIS_URL=...`. Provision `/run/keys/smallworld.env` with sops-nix, agenix,
systemd credentials, or another secret manager.

## Build and validation

Go dependencies are fetched reproducibly using the fixed `vendorHash` in
`packages.nix`. JavaScript dependencies are resolved reproducibly from each
`package-lock.json` with `importNpmLock`. Generated `vendor/`, `node_modules/`,
and `result` paths are intentionally not committed.

```console
nix flake check
nix build .#backend
nix build .#frontend
nix build .#map-generator
nix build .#checks.x86_64-linux.module --no-link
```

The last command builds the complete NixOS system closure without booting a VM
or modifying the host system. It works from a non-NixOS Linux host with Nix.

The map generator is a Git submodule. Clone with `--recurse-submodules`, or run
`git submodule update --init --recursive`, before building from a non-flake
working tree.
