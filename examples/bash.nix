{ pkgs, buildImage }:
buildImage {
  contents = [
    # When we want tools in /, we need to symlink them in order to
    # still have libraries in /nix/store. This differs from
    # dockerTools.buildImage but this allows to avoid habing files
    # both in / and /nix/store.
    (pkgs.symlinkJoin { name = "root"; paths = [ pkgs.bashInteractive pkgs.coreutils ]; })
  ];
  config = {
    Cmd = [ "/bin/bash" ];
  };
}
