{
  description = "A very basic flake";

  outputs = { self, nixpkgs }:
  let
    pkgs = nixpkgs.legacyPackages.x86_64-linux;

    containers-image-nix = pkgs.buildGoModule rec {
      pname = "container-images-nix";
      version = "0.0.1";
      src = pkgs.lib.cleanSourceWith {
        src = ./.;
        filter = path: type:
        let
          p = baseNameOf path;
        in !(p == "flake.nix" || p == "flake.lock" || p == "examples");
      };
      vendorSha256 = "sha256-gBme4IheJ/cJCRwRH3pnZlU7LKePD2eo7kiZldqQikY=";
    };

    skopeo = pkgs.skopeo.overrideAttrs (old: {
      preBuild = let
        patch = pkgs.fetchurl {
          url = "https://github.com/nlewo/image/commit/023556c0d31b155fd73e77ea8d06b7aee87adea8.patch";
          sha256 = "sha256-ygU9jrtxt0KnslWbqjU5fnmwvEfzkIRPjjjil/YiuwQ=";
        };
      in ''
        mkdir -p vendor/github.com/nlewo/nix2container/
        cp -r ${containers-image-nix.src}/* vendor/github.com/nlewo/nix2container/
        cd vendor/github.com/containers/image/v5
        mkdir nix/
        touch nix/transport.go
        patch -p1 < ${patch}
        cd -
      '';
    });

    pushToDockerDeamon = image: pkgs.writeScriptBin "push-to-docker-deamon" ''
      ${skopeo}/bin/skopeo --insecure-policy copy nix:${image} docker-daemon:${image.name}:${image.tag}
      echo Docker image ${image.name}:${image.tag} have been loaded
    '';

    buildLayer = {
      # A list of store paths to include in the layer
      deps,
      # A list of store paths to include in the layer root
      contents ? [],
      # A store path to exclude. This is mainly useful to exclude the
      # configuration file from the container layer.
      exclude ? null,
      # A list of layers containing dependencies: if a store path of the
      # currently built layer already belongs to a dependency layer,
      # this store path is skipped
      isolatedDeps ? []
    }: let
      rewrites = pkgs.lib.concatMapStringsSep " " (p: "--rewrite '${p},^${p},'") contents;
      allDeps = deps ++ contents;
    in
    pkgs.runCommand "layer.json" {} ''
      mkdir $out
      ${containers-image-nix}/bin/nix2container layers-from-reproducible-storepaths \
        ${pkgs.closureInfo {rootPaths = allDeps;}}/store-paths \
        ${rewrites} \
        ${pkgs.lib.concatMapStringsSep " "  (l: l + "/layer.json") isolatedDeps} \
        ${pkgs.lib.optionalString (exclude != null) "--exclude ${exclude}"} > $out/layer.json
      '';
  
    buildImage = {
      name,
      tag ? "latest",
      # An attribute set describing a container configuration
      config,
      isolatedDeps ? [],
      contents ? [],
    }:
      let
        configFile = pkgs.writeText "config.json" (builtins.toJSON config);
        # This layer contains all config dependencies. We exclude the
        # configFile because it is already part of the image, as a
        # specific blob.
        configDepsLayer = buildLayer {
          inherit contents;
          deps = [configFile];
          exclude = configFile;
          isolatedDeps = isolatedDeps;
        };
        layerPaths = pkgs.lib.concatMapStringsSep " " (l: l + "/layer.json") ([configDepsLayer] ++ isolatedDeps);
        image = pkgs.runCommand "image.json" {} ''
          echo ${containers-image-nix}/bin/nix2container image ${configFile} ${layerPaths}
          ${containers-image-nix}/bin/nix2container image ${configFile} ${layerPaths} > $out
        '';
        namedImage = image // { inherit name tag; };
      in namedImage // {
          pushToDockerDeamon = pushToDockerDeamon namedImage;
      };
    examples = {
      hello = import ./examples/hello.nix { inherit pkgs buildImage buildLayer; };
      nginx = import ./examples/nginx.nix { inherit pkgs buildImage buildLayer; };
      bash = import ./examples/bash.nix { inherit pkgs buildImage; };
      basic = import ./examples/basic.nix { inherit pkgs buildImage; };
      };
  in
  {
    packages.x86_64-linux = {
      inherit containers-image-nix examples skopeo;
    };
  };
}
