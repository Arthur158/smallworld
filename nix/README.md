# Smallworld on NixOS

This flake packages the gateway, room workers, map generator, and React UI. Its
NixOS module also configures nginx, PostgreSQL, Redis, service users, startup
ordering, persistence, firewall access, and systemd hardening.

## Fast local/LAN deployment

Add this repository to the target machine's flake inputs:

```nix
{
  inputs.smallworld.url = "github:YOUR-USER/smallworld";

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
