# The permissions on the /tmp directory are set to 777.
{ pkgs, nix2container }:

let
  tmp = pkgs.runCommand "tmp" {} ''
    mkdir -p $out/tmp
  '';
  openbar = nix2container.buildLayer {
    contents = [ tmp ];
    perms = [
      {
        path = tmp;
        regex = ".*";
        mode = "0777";
      }
    ];
  };
in

nix2container.buildImage {
  name = "openbar";
  isolatedDeps = [openbar];
  config = {
    entrypoint = ["${pkgs.coreutils}/bin/ls" "-l" "/"];
  };
}
