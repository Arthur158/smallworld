{
  description = "Smallworld packages and NixOS deployment module";

  inputs = {
    # Use the official channel archive instead of GitHub's API-backed flake
    # shorthand. This avoids unauthenticated GitHub API rate limits.
    nixpkgs.url = "https://channels.nixos.org/nixos-26.05/nixexprs.tar.xz";
    self.submodules = true;
  };

  outputs =
    { self, nixpkgs }:
    let
      supportedSystems = [
        "x86_64-linux"
        "aarch64-linux"
      ];
      forAllSystems = nixpkgs.lib.genAttrs supportedSystems;
    in
    {
      packages = forAllSystems (
        system:
        let
          pkgs = import nixpkgs { inherit system; };
          smallworld = pkgs.callPackage ./nix/packages.nix { src = self; };
        in
        {
          inherit (smallworld) backend frontend map-generator;
          default = smallworld.backend;
        }
      );

      checks = forAllSystems (
        system:
        let
          pkgs = import nixpkgs { inherit system; };
          smallworld = pkgs.callPackage ./nix/packages.nix { src = self; };
        in
        {
          inherit (smallworld) backend frontend map-generator;
          module = (nixpkgs.lib.nixosSystem {
            inherit system;
            modules = [
              self.nixosModules.default
              {
                boot.loader.grub.enable = false;
                fileSystems."/" = {
                  device = "none";
                  fsType = "tmpfs";
                };
                services.smallworld.enable = true;
                system.stateVersion = "26.05";
              }
            ];
          }).config.system.build.toplevel;
        }
      );

      devShells = forAllSystems (
        system:
        let
          pkgs = import nixpkgs { inherit system; };
        in
        {
          default = pkgs.mkShell {
            packages = with pkgs; [
              go
              nodejs_22
              postgresql
              redis
            ];
          };
        }
      );

      nixosModules = {
        default = import ./nix/module.nix;
        smallworld = self.nixosModules.default;
      };
    };
}
