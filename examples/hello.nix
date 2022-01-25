{ pkgs, nix2container }:
let
  application = pkgs.writeScript "conversation" ''
    ${pkgs.hello}/bin/hello 
    echo "Haaa aa... I'm dying!!!"
  '';
in
nix2container.buildImage {
  name = "hello";
  config = {
    entrypoint = ["${pkgs.bash}/bin/bash" application];
  };
  isolatedDeps = [
    (nix2container.buildLayer { deps = [pkgs.bash pkgs.hello]; })
  ];
}
