{ pkgs, nix2container }:
let
  test = pkgs.runCommand "test" { } ''
    mkdir -p $out/tmp
    touch $out/tmp/test1.txt
    touch $out/tmp/test2.txt
  '';
in nix2container.buildImage {
  name = "perms";
  config.entrypoint = "${pkgs.coreutils}/bin/ls -l /tmp/";
  copyToRoot = [ test ];
  perms = [
    {
      path = test;
      regex = "";
      mode = "0777";
    }
  ];
}
