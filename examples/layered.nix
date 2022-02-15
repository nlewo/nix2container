{ pkgs, nix2container }:
nix2container.buildImage {
  name = "layered";
  config = {
    entrypoint = ["${pkgs.hello}/bin/hello"];
  };
  maxLayers = 3;
}
