{ pkgs }: let
  alpine = pkgs.nix2container.pullImage {
    imageName = "alpine";
    imageDigest = "sha256:115731bab0862031b44766733890091c17924f9b7781b79997f5f163be262178";
    sha256 = "sha256-o4GvFCq6pvzASvlI5BLnk+Y4UN6qKL2dowuT0cp8q7Q=";
  };
in
pkgs.nix2container.buildImage {
  name = "from-image";
  fromImage = alpine;
  config = {
    entrypoint = [ "${pkgs.coreutils}/bin/ls" "-l" "/etc/alpine-release"];
  };
}
