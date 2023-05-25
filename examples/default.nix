{ pkgs, nix2container, skopeo-nix2container }: {
  hello = pkgs.callPackage ./hello.nix { inherit nix2container; };
  nginx = pkgs.callPackage ./nginx.nix { inherit nix2container; };
  bash = pkgs.callPackage ./bash.nix { inherit nix2container; };
  basic = pkgs.callPackage ./basic.nix { inherit nix2container; };
  nonReproducible = pkgs.callPackage ./non-reproducible.nix { inherit nix2container; };
  fromImage = pkgs.callPackage ./from-image.nix { inherit nix2container; };
  fromImageManifest = pkgs.callPackage ./from-image-manifest.nix { inherit nix2container; };
  uwsgi = pkgs.callPackage ./uwsgi { inherit nix2container; };
  openbar = pkgs.callPackage ./openbar.nix { inherit nix2container; };
  layered = pkgs.callPackage ./layered.nix { inherit nix2container; };
  nested = pkgs.callPackage ./nested.nix { inherit nix2container; };
  nix = pkgs.callPackage ./nix.nix { inherit nix2container; };
  nix-user = pkgs.callPackage ./nix-user.nix { inherit nix2container; };
  ownership = pkgs.callPackage ./ownership.nix { inherit nix2container; };

  update-manifests = let
    image = "library/alpine";
    skopeo = "${skopeo-nix2container}/bin/skopeo";
    jq = "${pkgs.jq}/bin/jq";
    filter = ''.manifests[] | select((.platform.os=="linux") and (.platform.architecture=="amd64")) | .digest'';
  in pkgs.writeShellScriptBin "update-manifests" ''
    set -e
    hash=$(${skopeo} inspect docker://${image} --raw | ${jq} -r '${filter}')
    ${skopeo} inspect docker://${image}@$hash --raw | ${jq} > examples/alpine-manifest.json
  '';
}
