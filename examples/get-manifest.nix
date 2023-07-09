{ nix2container }:
{
  images = {
    multiArch = nix2container.pullImageFromManifest {
      imageName = "library/alpine";
      os = "linux";
      arch = "amd64";
    };

    singleArch = nix2container.pullImageFromManifest {
      imageName = "rancher/systemd-node";
      imageTag = "v0.0.4";
    };

    quayio = nix2container.pullImageFromManifest {
      imageName = "containers/podman";
      imageTag = "v4.5";
      registryUrl = "quay.io";
    };
  };
}
