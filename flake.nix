{
  description = "nix2container";

  inputs.flake-utils.url = "github:numtide/flake-utils";
  inputs.nixpkgs.url = "github:NixOS/nixpkgs";

  outputs = { self, nixpkgs, flake-utils }:
  {
    overlay = import ./overlay.nix;
  } // (flake-utils.lib.eachDefaultSystem (system:
    let
      pkgs = import nixpkgs {
        inherit system;
        overlays = [ self.overlay ];
      };
      examples = import ./examples { inherit pkgs; };
    in
    rec {
      packages = {
        inherit (pkgs) containers-image-nix skopeo-nix2container;
        inherit examples;
      };
      defaultPackage = packages.containers-image-nix;
    }));
}
