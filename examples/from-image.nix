{ pkgs, nix2container }: let
  alpine = nix2container.pullImage {
    imageName = "alpine";
    imageDigest = "sha256:115731bab0862031b44766733890091c17924f9b7781b79997f5f163be262178";
    sha256 = "sha256-UHUcZBsoZ+DpAxroGPHkJ848sI9BF3iUia8RvUkfEC0=";
  };
in
nix2container.buildImage {
  name = "from-image";
  fromImage = alpine;
  config = {
    entrypoint = [ "${pkgs.coreutils}/bin/ls" "-l" "/etc/alpine-release"];
  };
}
