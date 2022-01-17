{ pkgs }:
let
  nonReproducible = pkgs.runCommand "non-reproducible" {} ''
    date > $out
  '';
in
pkgs.nix2container.buildImage {
  name = "non-reproducible";
  config = {
    entrypoint = ["${pkgs.coreutils}/bin/cat" "${nonReproducible}"];
  };
  isolatedDeps = [
    (pkgs.nix2container.buildLayer {
      deps = [nonReproducible];
      reproducible = false;
    })
  ];
}
