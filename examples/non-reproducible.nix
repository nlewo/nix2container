{pkgs, buildImage, buildLayer}:
let
  nonReproducible = pkgs.runCommand "non-reproducible" {} ''
    date > $out
  '';
in
buildImage {
  name = "non-reproducible";
  config = {
    entrypoint = ["${pkgs.coreutils}/bin/cat" "${nonReproducible}"];
  };
  isolatedDeps = [
    (buildLayer {
      deps = [nonReproducible];
      reproducible = false;
    })
  ];
}
