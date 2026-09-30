{ pkgs, nix2container }:
let
  test = pkgs.runCommand "test" { } ''
    mkdir -p $out/tmp/sub
    touch $out/tmp/test1.txt
    touch $out/tmp/sub/test2.txt
  '';
in nix2container.buildImage {
  name = "perms";
  config.entrypoint = "${pkgs.coreutils}/bin/ls -l /tmp/";
  copyToRoot = [ test ];
  skipCopyTo = "^/tmp/sub";
}
