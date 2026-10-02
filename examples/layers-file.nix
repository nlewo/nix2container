{ pkgs, nix2container }:

let
  # A split computed at build time: hello in the first layer, all of
  # its dependencies in the second one.
  split = pkgs.runCommand "split.json" {
    __structuredAttrs = true;
    exportReferencesGraph.graph = [ pkgs.hello ];
    nativeBuildInputs = [ pkgs.jq ];
  } ''
    jq --arg hello ${pkgs.hello} \
      '[[$hello], [.graph[].path | select(. != $hello)]]' \
      .attrs.json > $out
  '';
in
nix2container.buildImage {
  name = "layers-file";
  config = {
    entrypoint = ["${pkgs.hello}/bin/hello"];
  };
  layers = [(nix2container.buildLayer {
    deps = [pkgs.hello];
    layersFile = split;
  })];
}
