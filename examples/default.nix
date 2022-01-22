{ pkgs }: {
  hello = pkgs.callPackage ./hello.nix { };
  nginx = pkgs.callPackage ./nginx.nix { };
  bash = pkgs.callPackage ./bash.nix { };
  basic = pkgs.callPackage ./basic.nix { };
  nonReproducible = pkgs.callPackage ./non-reproducible.nix { };
  fromImage = pkgs.callPackage ./from-image.nix { };
}
