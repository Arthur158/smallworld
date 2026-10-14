{ inputs, ... }:
{
  imports = [ inputs.smallworld.nixosModules.default ];

  services.smallworld = {
    enable = true;
    hostName = "smallworld.example.com";
    port = 443;
    enableACME = true;
    workerCount = 2;
    allowedOrigins = [ "https://smallworld.example.com" ];
  };

  security.acme = {
    acceptTerms = true;
    defaults.email = "admin@example.com";
  };
}
