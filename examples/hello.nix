{pkgs, buildImage, buildLayer}:
let
  application = pkgs.writeScript "conversation" ''
    ${pkgs.hello}/bin/hello 
    echo "Haaa aa... I'm dying!!!"
  '';
in
buildImage {
  name = "hello";
  config = {
    entrypoint = ["${pkgs.bash}/bin/bash" application];
  };
  isolatedDeps = [
    (buildLayer { deps = [pkgs.bash pkgs.hello]; })
  ];
}
