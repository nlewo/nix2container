{ pkgs, nix2container }:
let
  nonReproducible = pkgs.runCommand "non-reproducible" {} ''
    date > $out
  '';
in
nix2container.buildImage {
  name = "non-reproducible";
  config = {
    entrypoint = ["${pkgs.coreutils}/bin/cat" "${nonReproducible}"];
  };
  layers = [
    (nix2container.buildLayer {
      deps = [nonReproducible];
      reproducible = false;
    })
  ];
}
