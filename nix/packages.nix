{
  lib,
  pkgs,
  src,
}:
let
  backendSrc = src + "/backend2";
  frontendSrc = src + "/frontend";
  mapGeneratorSrc = src + "/smallworld-map-generator";
in
rec {
  backend = pkgs.buildGoModule {
    pname = "smallworld-backend";
    version = "0.1.0";
    src = backendSrc;

    vendorHash = "sha256-0yrNVxm/+xbc9Pe2WWOhJNxhtKcRVKRqFv2RlXxkBlw=";
    subPackages = [
      "cmd/gateway"
      "cmd/roomworker"
    ];

    doCheck = true;

    meta = {
      description = "Smallworld websocket gateway and room worker";
      license = lib.licenses.mit;
      platforms = lib.platforms.linux;
    };
  };

  frontend = pkgs.buildNpmPackage {
    pname = "smallworld-frontend";
    version = "0.1.0";
    src = frontendSrc;

    npmDeps = pkgs.importNpmLock { npmRoot = frontendSrc; };
    npmConfigHook = pkgs.importNpmLock.npmConfigHook;
    npmBuildScript = "build";

    installPhase = ''
      runHook preInstall
      mkdir -p "$out/share/smallworld/frontend"
      cp -r build/. "$out/share/smallworld/frontend/"
      runHook postInstall
    '';

    meta = {
      description = "Static Smallworld React frontend";
      license = lib.licenses.mit;
      platforms = lib.platforms.linux;
    };
  };

  map-generator = pkgs.stdenv.mkDerivation {
    pname = "smallworld-map-generator";
    version = "0.1.0";
    src = mapGeneratorSrc;

    npmDeps = pkgs.importNpmLock { npmRoot = mapGeneratorSrc; };
    nativeBuildInputs = [
      pkgs.importNpmLock.npmConfigHook
      pkgs.makeWrapper
      pkgs.nodejs_22
    ];

    dontNpmBuild = true;

    installPhase = ''
      runHook preInstall
      app="$out/lib/smallworld-map-generator"
      mkdir -p "$app" "$out/bin"
      cp -r server src package.json "$app/"
      cp -rL node_modules "$app/node_modules"
      makeWrapper ${pkgs.nodejs_22}/bin/node "$out/bin/smallworld-map-generator" \
        --add-flags "$app/server/api-server.mjs"
      runHook postInstall
    '';

    meta = {
      description = "Smallworld procedural map generation API";
      license = lib.licenses.mit;
      platforms = lib.platforms.linux;
      mainProgram = "smallworld-map-generator";
    };
  };
}
