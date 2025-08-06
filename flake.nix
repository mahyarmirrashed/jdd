{
  description = "Flake for github:mahyarmirrashed/jdd";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-25.05";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs =
    {
      self,
      nixpkgs,
      flake-utils,
    }:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = nixpkgs.legacyPackages.${system};

        jdd = pkgs.buildGoModule {
          pname = "jdd";
          version = "dev";

          src = self;

          vendorHash = "sha256-MnqC6nHlthn+N3+T4xFpy5CVmk7mdSlw9DSPI2zAEUY=";
        };
      in
      {
        devShells.default = pkgs.mkShell {
          packages = with pkgs; [
            air
            go
            jdd
          ];
        };
      }
    );
}
