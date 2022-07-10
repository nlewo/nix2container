{ pkgs, nix2container }:
nix2container.buildImage {
  name = "bash";
  copyToRoot = [
    # When we want tools in /, we need to symlink them in order to
    # still have libraries in /nix/store. This behavior differs from
    # dockerTools.buildImage but this allows to avoid having files
    # in both / and /nix/store.
    (pkgs.buildEnv {
      name = "root";
      paths = [ pkgs.bashInteractive pkgs.coreutils ];
      pathsToLink = [ "/bin" ];
    })
  ];
  config = {
    Cmd = [ "/bin/bash" ];
  };
}
