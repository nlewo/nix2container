{ pkgs, nix2container }:
nix2container.buildImage {
  name = "metadata";
  config = {
    entrypoint = ["${pkgs.hello}/bin/hello"];
  };
  layers = [
    (nix2container.buildLayer {
      deps = [ pkgs.hello ];
      metadata = {
        created_by = "test created_by";
        author = "test author";
        comment = "test comment";
      };
    })
  ];
}
