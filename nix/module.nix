{
  config,
  lib,
  pkgs,
  ...
}:
let
  cfg = config.services.smallworld;
  packages = pkgs.callPackage ./packages.nix { src = ../.; };

  databaseUrl = "postgresql:///smallworld?host=/run/postgresql&sslmode=disable";
  redisAddress = "127.0.0.1:${toString cfg.redis.port}";
  virtualHostName = if cfg.hostName == null then "_" else cfg.hostName;

  commonEnvironment = {
    DATABASE_URL = if cfg.database.createLocally then databaseUrl else cfg.database.url;
    REDIS_ADDR = if cfg.redis.createLocally then redisAddress else cfg.redis.address;
    ROOM_WORKER_COUNT = toString cfg.workerCount;
  };

  commonServiceConfig = {
    User = "smallworld";
    Group = "smallworld";
    Restart = "on-failure";
    RestartSec = "2s";
    NoNewPrivileges = true;
    PrivateTmp = true;
    ProtectHome = true;
    ProtectSystem = "strict";
    ProtectKernelTunables = true;
    ProtectKernelModules = true;
    ProtectControlGroups = true;
    RestrictSUIDSGID = true;
  };

  localDependencies =
    lib.optional cfg.database.createLocally "postgresql-setup.service"
    ++ lib.optional cfg.redis.createLocally "redis-smallworld.service";

  mkWorker = workerId:
    lib.nameValuePair "smallworld-worker-${toString workerId}" {
      description = "Smallworld room worker ${toString workerId}";
      wantedBy = [ "multi-user.target" ];
      wants = localDependencies ++ [
        "smallworld-map-generator.service"
        "smallworld-gateway.service"
      ];
      after = [ "network.target" ] ++ localDependencies ++ [
        "smallworld-map-generator.service"
        "smallworld-gateway.service"
      ];
      environment = commonEnvironment // {
        WORKER_ID = toString workerId;
        HEALTH_ADDR = "127.0.0.1:${toString (cfg.workerHealthPortBase + workerId)}";
        MAP_GENERATOR_URL = "http://127.0.0.1:${toString cfg.mapGeneratorPort}";
      };
      serviceConfig =
        commonServiceConfig
        // {
          ExecStartPre = "${pkgs.curl}/bin/curl --fail --silent --retry 60 --retry-connrefused --retry-delay 1 http://127.0.0.1:8080/readyz";
          ExecStart = "${cfg.backendPackage}/bin/roomworker";
        }
        // lib.optionalAttrs (cfg.environmentFile != null) {
          EnvironmentFile = cfg.environmentFile;
        };
    };
in
{
  options.services.smallworld = {
    enable = lib.mkEnableOption "the Smallworld game stack";

    backendPackage = lib.mkPackageOption packages "backend" { };
    frontendPackage = lib.mkPackageOption packages "frontend" { };
    mapGeneratorPackage = lib.mkPackageOption packages "map-generator" { };

    hostName = lib.mkOption {
      type = with lib.types; nullOr str;
      default = null;
      example = "smallworld.example.com";
      description = "Public hostname. Null configures a catch-all virtual host.";
    };

    listenAddress = lib.mkOption {
      type = lib.types.str;
      default = "0.0.0.0";
      description = "Address on which nginx accepts public connections.";
    };

    port = lib.mkOption {
      type = lib.types.port;
      default = 3000;
      description = "Public HTTP port.";
    };

    openFirewall = lib.mkOption {
      type = lib.types.bool;
      default = true;
      description = "Open the public HTTP port in the NixOS firewall.";
    };

    enableACME = lib.mkOption {
      type = lib.types.bool;
      default = false;
      description = "Obtain and use an ACME certificate for hostName.";
    };

    workerCount = lib.mkOption {
      type = lib.types.ints.positive;
      default = 2;
      description = "Number of room workers. Do not change this while rooms are active.";
    };

    workerHealthPortBase = lib.mkOption {
      type = lib.types.port;
      default = 8100;
      description = "First loopback health port allocated to room workers.";
    };

    mapGeneratorPort = lib.mkOption {
      type = lib.types.port;
      default = 3001;
      description = "Loopback port used by the map generator API.";
    };

    allowedOrigins = lib.mkOption {
      type = lib.types.listOf lib.types.str;
      default = [ "*" ];
      example = [ "https://smallworld.example.com" ];
      description = "Origins accepted by the websocket gateway.";
    };

    environmentFile = lib.mkOption {
      type = with lib.types; nullOr path;
      default = null;
      example = "/run/keys/smallworld.env";
      description = "Optional systemd environment file for secret or external-service overrides.";
    };

    database = {
      createLocally = lib.mkOption {
        type = lib.types.bool;
        default = true;
        description = "Create a local PostgreSQL database and role using peer authentication.";
      };

      url = lib.mkOption {
        type = lib.types.str;
        default = "";
        example = "postgresql://smallworld@db.example.com/smallworld?sslmode=require";
        description = "External DATABASE_URL when createLocally is false. Prefer environmentFile for credentials.";
      };
    };

    redis = {
      createLocally = lib.mkOption {
        type = lib.types.bool;
        default = true;
        description = "Create a loopback-only persistent Redis instance.";
      };

      address = lib.mkOption {
        type = lib.types.str;
        default = "";
        example = "redis.example.com:6379";
        description = "External REDIS_ADDR when createLocally is false.";
      };

      port = lib.mkOption {
        type = lib.types.port;
        default = 6379;
        description = "Port for the local Redis instance.";
      };
    };
  };

  config = lib.mkIf cfg.enable {
    assertions = [
      {
        assertion = cfg.database.createLocally || cfg.database.url != "" || cfg.environmentFile != null;
        message = "Set services.smallworld.database.url or environmentFile when local PostgreSQL is disabled.";
      }
      {
        assertion = cfg.redis.createLocally || cfg.redis.address != "" || cfg.environmentFile != null;
        message = "Set services.smallworld.redis.address or environmentFile when local Redis is disabled.";
      }
      {
        assertion = !cfg.enableACME || cfg.hostName != null;
        message = "services.smallworld.hostName is required when enableACME is true.";
      }
      {
        assertion = cfg.workerHealthPortBase + cfg.workerCount - 1 <= 65535;
        message = "Smallworld worker health ports exceed 65535.";
      }
    ];

    users.groups.smallworld = { };
    users.users.smallworld = {
      isSystemUser = true;
      group = "smallworld";
      description = "Smallworld services";
    };

    services.postgresql = lib.mkIf cfg.database.createLocally {
      enable = true;
      ensureDatabases = [ "smallworld" ];
      ensureUsers = [
        {
          name = "smallworld";
          ensureDBOwnership = true;
        }
      ];
    };

    services.redis.servers.smallworld = lib.mkIf cfg.redis.createLocally {
      enable = true;
      bind = "127.0.0.1";
      port = cfg.redis.port;
      openFirewall = false;
      appendOnly = true;
    };

    services.nginx = {
      enable = true;
      recommendedGzipSettings = true;
      recommendedOptimisation = true;
      recommendedProxySettings = true;
      recommendedTlsSettings = cfg.enableACME;

      virtualHosts.${virtualHostName} = {
        default = cfg.hostName == null;
        enableACME = cfg.enableACME;
        forceSSL = cfg.enableACME;
        root = "${cfg.frontendPackage}/share/smallworld/frontend";
        listen = [
          {
            addr = cfg.listenAddress;
            port = cfg.port;
            ssl = cfg.enableACME;
          }
        ];

        locations."/ws" = {
          proxyPass = "http://127.0.0.1:8080/ws";
          proxyWebsockets = true;
          extraConfig = "proxy_read_timeout 86400;";
        };

        locations."/generator-api/".proxyPass =
          "http://127.0.0.1:${toString cfg.mapGeneratorPort}/api/";

        locations."/" = {
          tryFiles = "$uri $uri/ /index.html";
        };
      };
    };

    networking.firewall.allowedTCPPorts = lib.optionals cfg.openFirewall [ cfg.port ];

    systemd.services =
      {
        smallworld-map-generator = {
          description = "Smallworld map generator API";
          wantedBy = [ "multi-user.target" ];
          after = [ "network.target" ];
          environment = {
            HOST = "127.0.0.1";
            PORT = toString cfg.mapGeneratorPort;
            NODE_ENV = "production";
          };
          serviceConfig =
            commonServiceConfig
            // {
              ExecStart = "${cfg.mapGeneratorPackage}/bin/smallworld-map-generator";
              WorkingDirectory = "${cfg.mapGeneratorPackage}/lib/smallworld-map-generator";
            }
            // lib.optionalAttrs (cfg.environmentFile != null) {
              EnvironmentFile = cfg.environmentFile;
            };
        };

        smallworld-gateway = {
          description = "Smallworld websocket gateway";
          wantedBy = [ "multi-user.target" ];
          wants = localDependencies;
          after = [ "network.target" ] ++ localDependencies;
          before = [ "nginx.service" ];
          environment = commonEnvironment // {
            LISTEN_ADDR = "127.0.0.1:8080";
            GATEWAY_ID = "${config.networking.hostName}-gateway";
            WS_ALLOWED_ORIGINS = lib.concatStringsSep "," cfg.allowedOrigins;
          };
          serviceConfig =
            commonServiceConfig
            // {
              ExecStart = "${cfg.backendPackage}/bin/gateway";
            }
            // lib.optionalAttrs (cfg.environmentFile != null) {
              EnvironmentFile = cfg.environmentFile;
            };
        };

        nginx = {
          wants = [
            "smallworld-gateway.service"
            "smallworld-map-generator.service"
          ];
          after = [
            "smallworld-gateway.service"
            "smallworld-map-generator.service"
          ];
        };
      }
      // builtins.listToAttrs (map mkWorker (lib.range 0 (cfg.workerCount - 1)));
  };
}
