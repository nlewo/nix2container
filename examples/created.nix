{ pkgs, nix2container }:
nix2container.buildImage {
  name = "created";
  config = {
    entrypoint = ["${pkgs.hello}/bin/hello"];
  };
  created = "2024-05-13T09:31:10Z";
}
