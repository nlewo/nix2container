{
  description = "nix2container";

  inputs.flake-utils.url = "github:numtide/flake-utils";
  inputs.nixpkgs.url = "github:NixOS/nixpkgs";

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
        nix2container = import ./. {
          inherit pkgs system;
        };
        examples = import ./examples {
          inherit pkgs;
          inherit (nix2container) nix2container;
        };
        tests = import ./tests {
          inherit pkgs examples;
          inherit (nix2container) nix2container;
        };
      in
        rec {
          packages = {
            inherit (nix2container) nix2containerUtil skopeo-nix2container nix2container;
            inherit examples tests;
          };
          defaultPackage = packages.nix2containerUtil;
        });
}
