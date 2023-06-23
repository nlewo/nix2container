{ pkgs, nix2container }: let
  # nix run .#examples.fromImageManifest.fromImage.getManifest > examples/alpine-manifest.json
  alpine = nix2container.pullImageFromManifest {
    imageName = "library/alpine";
    imageManifest = ./alpine-manifest.json;

    # These attributes aren't checked against the manifest; they are only
    # used to populate the supplied getManifest script.
    imageTag = "latest";
    os = "linux";
    arch = "amd64";
  };
in
nix2container.buildImage {
  name = "from-image-manifest";
  fromImage = alpine;
  config = {
    entrypoint = [ "${pkgs.coreutils}/bin/ls" "-l" "/etc/alpine-release"];
  };
}
