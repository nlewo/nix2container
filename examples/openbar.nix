# The permissions on the /tmp directory are set to 777.
{ pkgs, nix2container }:

let
  tmp = pkgs.runCommand "tmp" {} ''
    mkdir -p $out/tmp
  '';
in

nix2container.buildImage {
  name = "openbar";
  copyToRoot = [ tmp ];
  perms = [
    {
      path = tmp;
      regex = ".*";
      mode = "0777";
      }
  ];
  config = {
    entrypoint = ["${pkgs.coreutils}/bin/ls" "-l" "/"];
  };
}
