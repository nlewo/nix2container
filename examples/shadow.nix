{
  pkgs,
  nix2container,
}:
nix2container.buildImage {
  name = "shadow";
  tag = "latest";

  layers = [
    (nix2container.layers.shadow {includeRoot = true;})
  ];

  copyToRoot = [pkgs.coreutils];

  config = {
    User = "somebody";
  };
}
