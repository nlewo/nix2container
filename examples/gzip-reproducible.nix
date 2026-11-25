{ pkgs, nix2container }:
nix2container.buildImage {
  name = "gzip-reproducible";
  compressor = "gzip";
  config.entrypoint = [ "${pkgs.hello}/bin/hello" ];
  layers = [
    (nix2container.buildLayer {
      deps = [ pkgs.hello ];
      compressor = "gzip";
    })
  ];
}
