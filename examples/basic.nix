{pkgs, buildImage}:
buildImage {
  name = "basic";
  config = {
    entrypoint = ["${pkgs.hello}/bin/hello"];
  };
}
