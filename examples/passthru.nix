{ pkgs, nix2container }:
nix2container.buildImage {
  name = "passthru";
  config.entrypoint = ["${pkgs.hello}/bin/hello"];
  passthru.sampleScript =
    pkgs.writeShellApplication {
      name = "passthru-script";
      text = ''
        echo "Hello from passthru script!"
      '';
    };
}
