{ pkgs, nix2container }: let
  alpine = nix2container.pullImageByManifest {
    imageName = "library/alpine";
    # nix run .#examples.fromImageManifest.fromImage.getManifest > examples/alpine-manifest.json
    os = "linux";
    arch = "amd64";
    imageManifest = ./alpine-manifest.json;
  };
in
nix2container.buildImage {
  name = "from-image-manifest";
  fromImage = alpine;
  config = {
    entrypoint = [ "${pkgs.coreutils}/bin/ls" "-l" "/etc/alpine-release"];
  };
}
