{ pkgs, nix2container }: let
  alpine = nix2container.pullImageByManifest {
    imagePath = "library/alpine";
    # nix run .#examples.update-manifests to update this to the latest.
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
