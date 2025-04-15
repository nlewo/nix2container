{
  description = "nix2container: build container image with Nix";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs";

  outputs = { self, nixpkgs }:
    let
      defaultSystems = [
        "x86_64-linux"
        "aarch64-linux"
        "x86_64-darwin"
        "aarch64-darwin"
      ];
      perSystem = nixpkgs.lib.genAttrs defaultSystems;
    in {
      packages = perSystem (system:
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
        in {
          inherit (nix2container) nix2container-bin skopeo-nix2container nix2container;
          inherit examples tests;
          default = nix2container.nix2container-bin;
        }
      );
    };
}
