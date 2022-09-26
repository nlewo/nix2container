{ pkgs, nix2container }:
let
  test = pkgs.runCommand "test" { } ''
    mkdir -p $out/tmp
    touch $out/tmp/test1.txt
    touch $out/tmp/test2.txt
  '';
in
nix2container.buildImage {
  name = "ownership";
  config.entrypoint = [ "${pkgs.coreutils}/bin/ls" "-l" "/tmp/" ];
  copyToRoot = [ test ];
  perms = [
    {
      path = test;
      regex = "/tmp/test1.txt";
      uid = 1001;
      gid = 1001;
    }
  ];
}
