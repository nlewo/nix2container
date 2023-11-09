{
  description = "nix2container: build container image with Nix";

  inputs.flake-utils.url = "github:numtide/flake-utils";
  inputs.nixpkgs.url = "github:NixOS/nixpkgs";

  outputs = inputs:
    inputs.flake-utils.lib.eachDefaultSystem (system:
      let
        overlays.default = final: prev: {
          nix2container = import ./. { system = final.system; pkgs = prev; };
        };
        
        pkgs = inputs.nixpkgs.legacyPackages.${system}.extend overlays.default;
        
        examples = pkgs.callPackage ./examples { };
        tests = pkgs.callPackage ./tests { inherit examples; };

        packages = {
          inherit (pkgs.nix2container) nix2container-bin skopeo-nix2container;
          inherit examples tests;
          default = packages.nix2container-bin;
          hello = pkgs.nix2container.buildImage {
            name = "hello";
            config = {
              entrypoint = [ "${pkgs.hello}/bin/hello" ];
            };
          };
        };
      in
      {
        inherit packages;
        inherit (pkgs) nix2container;
        inherit overlays;
      });
}
