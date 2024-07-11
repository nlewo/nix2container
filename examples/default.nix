{ pkgs }: {
  hello = pkgs.callPackage ./hello.nix { };
  nginx = pkgs.callPackage ./nginx.nix { };
  bash = pkgs.callPackage ./bash.nix { };
  basic = pkgs.callPackage ./basic.nix { };
  nonReproducible = pkgs.callPackage ./non-reproducible.nix { };
  fromImage = pkgs.callPackage ./from-image.nix { };
  fromImageManifest = pkgs.callPackage ./from-image-manifest.nix { };
  getManifest = pkgs.callPackage ./get-manifest.nix { };
  uwsgi = pkgs.callPackage ./uwsgi { };
  openbar = pkgs.callPackage ./openbar.nix { };
  layered = pkgs.callPackage ./layered.nix { };
  nested = pkgs.callPackage ./nested.nix { };
  nix = pkgs.callPackage ./nix.nix { };
  nix-user = pkgs.callPackage ./nix-user.nix { };
  ownership = pkgs.callPackage ./ownership.nix { };
}
