{ pkgs, nix2container }: {
  hello = pkgs.callPackage ./hello.nix { inherit nix2container; };
  nginx = pkgs.callPackage ./nginx.nix { inherit nix2container; };
  bash = pkgs.callPackage ./bash.nix { inherit nix2container; };
  basic = pkgs.callPackage ./basic.nix { inherit nix2container; };
  nonReproducible = pkgs.callPackage ./non-reproducible.nix { inherit nix2container; };
  fromImage = pkgs.callPackage ./from-image.nix { inherit nix2container; };
  uwsgi = pkgs.callPackage ./uwsgi { inherit nix2container; };
}
