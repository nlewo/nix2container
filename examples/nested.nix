{ pkgs, nix2container }:
nix2container.buildImage {
  name = "nested";
  config = {
    entrypoint = ["${pkgs.hello}/bin/hello"];
  };
  layers = [(nix2container.buildLayer {
    deps = [pkgs.bashInteractive];
    layers = [
      (nix2container.buildLayer {
        deps = [pkgs.readline81];
      })
    ];
  })];
}
