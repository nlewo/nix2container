{ pkgs }:
pkgs.nix2container.buildImage {
  name = "basic";
  config = {
    entrypoint = ["${pkgs.hello}/bin/hello"];
  };
}
