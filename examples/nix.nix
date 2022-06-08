{ pkgs, nix2container }:
nix2container.buildImage {
  name = "nix";
  initializeNixDatabase = true;
  contents = [
    # nix-store uses cat program to display results as specified by
    # the image env variable NIX_PAGER.
    (pkgs.symlinkJoin { name = "root"; paths = [ pkgs.coreutils pkgs.nix pkgs.bash ]; })
  ];
  config = {
    Env = [
      "NIX_PAGER=cat"
      # A user is required by nix
      # https://github.com/NixOS/nix/blob/9348f9291e5d9e4ba3c4347ea1b235640f54fd79/src/libutil/util.cc#L478
      "USER=nobody"
    ];
  };
  # This is to check store path in nested layers are also added to the
  # Nix database.
  layers = [
    (nix2container.buildLayer {
      deps = [ pkgs.hello ];
    })
  ];
}
