{ pkgs, nix2container }:
nix2container.buildImage {
  name = "nested";
  config = {
    entrypoint = ["/bin/bash" "-c" "${pkgs.hello}/bin/hello"];
  };
  layers = [(nix2container.buildLayer {
    copyToRoot = pkgs.buildEnv {
      name = "root";
      paths = [pkgs.bashInteractive];
      pathsToLink = ["/bin"];
    };
    layers = [
      (nix2container.buildLayer {
        deps = [pkgs.readline];
      })
    ];
  })];
}
