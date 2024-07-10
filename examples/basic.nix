{ pkgs, nix2container }:
nix2container.buildImage {
  name = "basic";
  verifyTrace = true;
  config = {
    entrypoint = ["${pkgs.hello}/bin/hello"];
  };
}
