{ pkgs, nix2container }:
let
  nonReproducible = pkgs.runCommand "non-reproducible" {} ''
    echo -n "A non reproducible image built the " > $out
    date >> $out
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
